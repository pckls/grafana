package grafanaschema

RegistryItem: {
	id:           string
	name:         string
	description?: string
	aliasIds?: [string]
	excludeFromPicker?: bool
} @cuetsy(targetType="interface")
