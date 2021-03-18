import { DataSourceInstanceSettings, DataSourceJsonData } from '@grafana/data';
import { config } from '@grafana/runtime';

export function getAllDataSources(): Array<DataSourceInstanceSettings<DataSourceJsonData>> {
  return Object.values(config.datasources);
}

export function getDatasourceByName(name: string): DataSourceInstanceSettings<DataSourceJsonData> | undefined {
  return getAllDataSources().find((source) => source.name === name);
}

export function getPromAndLokiDataSources(): Array<DataSourceInstanceSettings<DataSourceJsonData>> {
  return getAllDataSources().filter(({ type }) => type === 'prometheus' || type === 'loki');
}
