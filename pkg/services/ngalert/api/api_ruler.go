package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/grafana/grafana/pkg/services/ngalert/store"

	apimodels "github.com/grafana/alerting-api/pkg/api"
	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/models"
	ngmodels "github.com/grafana/grafana/pkg/services/ngalert/models"
	"github.com/grafana/grafana/pkg/util"
	"github.com/prometheus/common/model"
)

type RulerSrv struct {
	store store.RuleStore
	log   log.Logger
}

func (srv RulerSrv) RouteDeleteNamespaceRulesConfig(c *models.ReqContext) response.Response {
	namespace := c.Params(":Namespace")
	namespaceUID, err := srv.store.GetNamespaceUIDBySlug(namespace, c.SignedInUser.OrgId, c.SignedInUser)
	if err != nil {
		return response.Error(http.StatusInternalServerError, fmt.Sprintf("failed to get namespace: %s", namespace), err)
	}
	if err := srv.store.DeleteNamespaceAlertRules(&ngmodels.DeleteNamespaceAlertRulesCommand{
		OrgID:        c.SignedInUser.OrgId,
		NamespaceUID: namespaceUID,
	}); err != nil {
		return response.Error(http.StatusInternalServerError, "failed to delete namespace alert rules", err)
	}
	return response.JSON(http.StatusAccepted, util.DynMap{"message": "namespace rules deleted"})
}

func (srv RulerSrv) RouteDeleteRuleGroupConfig(c *models.ReqContext) response.Response {
	namespace := c.Params(":Namespace")
	namespaceUID, err := srv.store.GetNamespaceUIDBySlug(namespace, c.SignedInUser.OrgId, c.SignedInUser)
	if err != nil {
		return response.Error(http.StatusInternalServerError, fmt.Sprintf("failed to get namespace: %s", namespace), err)
	}
	ruleGroup := c.Params(":Groupname")
	if err := srv.store.DeleteRuleGroupAlertRules(&ngmodels.DeleteRuleGroupAlertRulesCommand{
		OrgID:        c.SignedInUser.OrgId,
		NamespaceUID: namespaceUID,
		RuleGroup:    ruleGroup,
	}); err != nil {
		return response.Error(http.StatusInternalServerError, "failed to delete group alert rules", err)
	}
	return response.JSON(http.StatusAccepted, util.DynMap{"message": "rule group deleted"})
}

func (srv RulerSrv) RouteGetNamespaceRulesConfig(c *models.ReqContext) response.Response {
	namespace := c.Params(":Namespace")
	namespaceUID, err := srv.store.GetNamespaceUIDBySlug(namespace, c.SignedInUser.OrgId, c.SignedInUser)
	if err != nil {
		return response.Error(http.StatusInternalServerError, fmt.Sprintf("failed to get namespace: %s", namespace), err)
	}

	q := ngmodels.ListNamespaceAlertRulesQuery{
		OrgID:        c.SignedInUser.OrgId,
		NamespaceUID: namespaceUID,
	}
	if err := srv.store.GetNamespaceAlertRules(&q); err != nil {
		return response.Error(http.StatusInternalServerError, "failed to update rule group", err)
	}

	result := apimodels.NamespaceConfigResponse{}
	ruleGroupConfigs := make(map[string]apimodels.RuleGroupConfig)
	for _, r := range q.Result {
		ruleGroupConfig, ok := ruleGroupConfigs[r.RuleGroup]
		switch ok {
		case false:
			ruleGroupInterval := model.Duration(time.Duration(r.IntervalSeconds) * time.Second)
			ruleGroupConfigs[r.RuleGroup] = apimodels.RuleGroupConfig{
				Name:     r.RuleGroup,
				Interval: ruleGroupInterval,
				Rules: []apimodels.ExtendedRuleNode{
					toExtendedRuleNode(*r),
				},
			}
		case true:
			ruleGroupConfig.Rules = append(ruleGroupConfig.Rules, toExtendedRuleNode(*r))
			ruleGroupConfigs[r.RuleGroup] = ruleGroupConfig
		}
	}

	for _, ruleGroupConfig := range ruleGroupConfigs {
		result[namespace] = append(result[namespace], ruleGroupConfig)
	}

	return response.JSON(http.StatusAccepted, result)
}

func (srv RulerSrv) RouteGetRulegGroupConfig(c *models.ReqContext) response.Response {
	namespace := c.Params(":Namespace")
	namespaceUID, err := srv.store.GetNamespaceUIDBySlug(namespace, c.SignedInUser.OrgId, c.SignedInUser)
	if err != nil {
		return response.Error(http.StatusInternalServerError, fmt.Sprintf("failed to get namespace: %s", namespace), err)
	}

	ruleGroup := c.Params(":Groupname")
	q := ngmodels.ListRuleGroupAlertRulesQuery{
		OrgID:        c.SignedInUser.OrgId,
		NamespaceUID: namespaceUID,
		RuleGroup:    ruleGroup,
	}
	if err := srv.store.GetRuleGroupAlertRules(&q); err != nil {
		return response.Error(http.StatusInternalServerError, "failed to get group alert rules", err)
	}

	var ruleGroupInterval model.Duration
	ruleNodes := make([]apimodels.ExtendedRuleNode, 0, len(q.Result))
	for _, r := range q.Result {
		ruleGroupInterval = model.Duration(time.Duration(r.IntervalSeconds) * time.Second)
		ruleNodes = append(ruleNodes, toExtendedRuleNode(*r))
	}

	result := apimodels.RuleGroupConfigResponse{
		RuleGroupConfig: apimodels.RuleGroupConfig{
			Name:     ruleGroup,
			Interval: ruleGroupInterval,
			Rules:    ruleNodes,
		},
	}
	return response.JSON(http.StatusAccepted, result)
}

func (srv RulerSrv) RouteGetRulesConfig(c *models.ReqContext) response.Response {
	q := ngmodels.ListAlertRulesQuery{
		OrgID: c.SignedInUser.OrgId,
	}
	if err := srv.store.GetOrgAlertRules(&q); err != nil {
		return response.Error(http.StatusInternalServerError, "failed to get alert rules", err)
	}

	configs := make(map[string]map[string]apimodels.RuleGroupConfig)
	for _, r := range q.Result {
		namespace, err := srv.store.GetNamespaceByUID(r.NamespaceUID, c.SignedInUser.OrgId, c.SignedInUser)
		if err != nil {
			return response.Error(http.StatusInternalServerError, fmt.Sprintf("failed to get namespace: %s", r.NamespaceUID), err)
		}
		_, ok := configs[namespace]
		switch ok {
		case false:
			ruleGroupInterval := model.Duration(time.Duration(r.IntervalSeconds) * time.Second)
			configs[namespace] = make(map[string]apimodels.RuleGroupConfig)
			configs[namespace][r.RuleGroup] = apimodels.RuleGroupConfig{
				Name:     r.RuleGroup,
				Interval: ruleGroupInterval,
				Rules: []apimodels.ExtendedRuleNode{
					toExtendedRuleNode(*r),
				},
			}
		case true:
			ruleGroupConfig, ok := configs[namespace][r.RuleGroup]
			switch ok {
			case false:
				ruleGroupInterval := model.Duration(time.Duration(r.IntervalSeconds) * time.Second)
				configs[namespace][r.RuleGroup] = apimodels.RuleGroupConfig{
					Name:     r.RuleGroup,
					Interval: ruleGroupInterval,
					Rules: []apimodels.ExtendedRuleNode{
						toExtendedRuleNode(*r),
					},
				}
			case true:
				ruleGroupConfig.Rules = append(ruleGroupConfig.Rules, toExtendedRuleNode(*r))
				configs[namespace][r.RuleGroup] = ruleGroupConfig
			}
		}
	}

	result := apimodels.NamespaceConfigResponse{}
	for namespace, m := range configs {
		for _, ruleGroupConfig := range m {
			result[namespace] = append(result[namespace], ruleGroupConfig)
		}
	}
	return response.JSON(http.StatusAccepted, result)
}

func (srv RulerSrv) RoutePostNameRulesConfig(c *models.ReqContext, ruleGroupConfig apimodels.RuleGroupConfig) response.Response {
	namespace := c.Params(":Namespace")
	namespaceUID, err := srv.store.GetNamespaceUIDBySlug(namespace, c.SignedInUser.OrgId, c.SignedInUser)
	if err != nil {
		return response.Error(http.StatusInternalServerError, fmt.Sprintf("failed to get namespace: %s", namespace), err)
	}

	// TODO check permissions
	// TODO check quota
	// TODO validate UID uniqueness in the payload

	ruleGroup := ruleGroupConfig.Name

	if err := srv.store.UpdateRuleGroup(store.UpdateRuleGroupCmd{
		OrgID:           c.SignedInUser.OrgId,
		NamespaceUID:    namespaceUID,
		RuleGroup:       ruleGroup,
		RuleGroupConfig: ruleGroupConfig,
	}); err != nil {
		return response.Error(http.StatusInternalServerError, "failed to update rule group", err)
	}

	return response.JSON(http.StatusAccepted, util.DynMap{"message": "rule group updated successfully"})
}

func toExtendedRuleNode(r ngmodels.AlertRule) apimodels.ExtendedRuleNode {
	return apimodels.ExtendedRuleNode{
		GrafanaManagedAlert: &apimodels.ExtendedUpsertAlertDefinitionCommand{
			NoDataState:         apimodels.NoDataState(r.NoDataState),
			ExecutionErrorState: apimodels.ExecutionErrorState(r.ExecErrState),
			UpdateAlertDefinitionCommand: ngmodels.UpdateAlertDefinitionCommand{
				UID:       r.UID,
				OrgID:     r.OrgID,
				Title:     r.Title,
				Condition: r.Condition,
				Data:      r.Data,
			},
		},
	}
}
