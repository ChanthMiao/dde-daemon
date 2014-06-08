package main

// file header
const fileHeader = `// This file is automatically generated, please don't edit manually.
package network

import (
	"fmt"
)
`

// get key type
const tplGetKeyType = `
// Get key type
func get{{.Name | ToSectionFuncBaseName}}KeyType(key string) (t ktype) {
	switch key {
	default:
		t = ktypeUnknown{{range .Keys}}
	case {{.Name}}:
		t = {{.Type}}{{end}}
	}
	return
}
`

// check is key in current section
const tplIsKeyInSettingSection = `
// Check is key in current setting section
func isKeyIn{{.Name | ToSectionFuncBaseName}}(key string) bool {
	switch key { {{range .Keys}}{{if .UsedByBackEnd}}
	case {{.Name}}:
		return true{{end}}{{end}}
	}
	return false
}
`

// Ensure section and key exists and not empty
const tplEnsureNoEmpty = `{{$sectionFuncBaseName := .Name | ToSectionFuncBaseName}}{{$sectionName := .Name}}
// Ensure section and key exists and not empty
func ensureSection{{$sectionFuncBaseName}}Exists(data connectionData, errs sectionErrors, relatedKey string) {
	if !isSettingSectionExists(data, {{.Name}}) {
		rememberError(errs, relatedKey, {{.Name}}, fmt.Sprintf(NM_KEY_ERROR_MISSING_SECTION, {{.Name}}))
	}
	sectionData, _ := data[{{.Name}}]
	if len(sectionData) == 0 {
		rememberError(errs, relatedKey, {{.Name}}, fmt.Sprintf(NM_KEY_ERROR_EMPTY_SECTION, {{.Name}}))
	}
}{{range $i, $key := .Keys}}{{if $key.UsedByBackEnd}}{{$keyFuncBaseName := $key.Name | ToKeyFuncBaseName}}
func ensure{{$keyFuncBaseName}}NoEmpty(data connectionData, errs sectionErrors) {
	if !is{{$keyFuncBaseName}}Exists(data) {
		rememberError(errs, {{$sectionName}}, {{$key.Name}}, NM_KEY_ERROR_MISSING_VALUE)
	}{{if IfNeedCheckValueLength $key.Type}}
	value := get{{$keyFuncBaseName}}(data)
	if len(value) == 0 {
		rememberError(errs, {{$sectionName}}, {{$key.Name}}, NM_KEY_ERROR_EMPTY_VALUE)
	}{{end}}
}{{end}}{{end}}
`

// get key's default value
const tplGetDefaultValue = `{{$sectionFuncBaseName := .Name | ToSectionFuncBaseName}}
// Get key's default value
func get{{$sectionFuncBaseName}}DefaultValue(key string) (value interface{}) {
	switch key {
	default:
		logger.Error("invalid key:", key){{range .Keys}}{{if .UsedByBackEnd}}{{$default := ToKeyDefaultValue .Name}}
	case {{.Name}}:
		value = {{$default}}{{end}}{{end}}
	}
	return
}
`

// get json value generally
const tplGeneralGetterJSON = `{{$sectionFuncBaseName := .Name | ToSectionFuncBaseName}}
// Get JSON value generally
func generalGet{{$sectionFuncBaseName}}KeyJSON(data connectionData, key string) (value string) {
	switch key {
	default:
		logger.Error("generalGet{{$sectionFuncBaseName}}KeyJSON: invalide key", key){{range .Keys}}{{if .UsedByBackEnd}}
	case {{.Name}}:
		value = get{{.Name | ToKeyFuncBaseName}}JSON(data){{end}}{{end}}
	}
	return
}
`

// set json value generally
const tplGeneralSetterJSON = `{{$sectionFuncBaseName := .Name | ToSectionFuncBaseName}}
// Set JSON value generally
func generalSet{{$sectionFuncBaseName}}KeyJSON(data connectionData, key, valueJSON string) (err error) {
	switch key {
	default:
		logger.Error("generalSet{{$sectionFuncBaseName}}KeyJSON: invalide key", key){{range .Keys}}{{if .UsedByBackEnd}}
	case {{.Name}}:
		err = {{if .LogicSet}}logicSet{{else}}set{{end}}{{.Name | ToKeyFuncBaseName}}JSON(data, valueJSON){{end}}{{end}}
	}
	return
}
`

// check if key exists
const tplCheckExists = `
// Check if key exists{{$sectionName := .Name}}{{range $i, $key := .Keys}}{{if $key.UsedByBackEnd}}
func is{{$key.Name | ToKeyFuncBaseName}}Exists(data connectionData) bool {
	return isSettingKeyExists(data, {{$sectionName}}, {{$key.Name}})
}{{end}}{{end}}
`

// getter
const tplGetter = `
// Getter{{$sectionName := .Name}}{{range $i, $key := .Keys}}{{if $key.UsedByBackEnd}}{{$keyFuncBaseName := $key.Name | ToKeyFuncBaseName}}{{$realType := $key.Type | ToKeyTypeRealData}}
func get{{$keyFuncBaseName}}(data connectionData) (value {{$key.Type | ToKeyTypeRealData}}) {
	ivalue := getSettingKey(data, {{$sectionName}}, {{$key.Name}})
	value = {{$key.Type | ToKeyTypeInterfaceConverter}}(ivalue)
	return
}{{end}}{{end}}
`

// setter
const tplSetter = `
// Setter{{$sectionName := .Name}}{{range $i, $key := .Keys}}{{if $key.UsedByBackEnd}}
func set{{$key.Name | ToKeyFuncBaseName}}(data connectionData, value {{$key.Type | ToKeyTypeRealData}}) {
	setSettingKey(data, {{$sectionName}}, {{$key.Name}}, value)
}{{end}}{{end}}
`

// json getter
const tplJSONGetter = `
// JSON Getter{{$sectionName := .Name}}{{range $i, $key := .Keys}}{{if $key.UsedByBackEnd}}
func get{{$key.Name | ToKeyFuncBaseName}}JSON(data connectionData) (valueJSON string) {
	valueJSON = getSettingKeyJSON(data, {{$sectionName}}, {{$key.Name}}, get{{$sectionName | ToSectionFuncBaseName}}KeyType({{$key.Name}}))
	return
}{{end}}{{end}}
`

// json setter
const tplJSONSetter = `
// JSON Setter{{$sectionName := .Name}}{{range $i, $key := .Keys}}{{if $key.UsedByBackEnd}}
func set{{$key.Name | ToKeyFuncBaseName}}JSON(data connectionData, valueJSON string) (err error) {
	return setSettingKeyJSON(data, {{$sectionName}}, {{$key.Name}}, valueJSON, get{{$sectionName | ToSectionFuncBaseName}}KeyType({{$key.Name}}))
}{{end}}{{end}}
`

// logic json setter
const tplLogicJSONSetter = `
// Logic JSON Setter{{range $i, $key := .Keys}}{{if $key.LogicSet}}{{$keyFuncBaseName := $key.Name | ToKeyFuncBaseName}}
func logicSet{{$keyFuncBaseName}}JSON(data connectionData, valueJSON string) (err error) {
	err = set{{$keyFuncBaseName}}JSON(data, valueJSON)
	if err != nil {
		return
	}
	if is{{$keyFuncBaseName}}Exists(data) {
		value := get{{$keyFuncBaseName}}(data)
		err = logicSet{{$keyFuncBaseName}}(data, value)
	}
	return
}{{end}}{{end}}
`

// remover
const tplRemover = `
// Remover{{$sectionName := .Name}}{{range $i, $key := .Keys}}{{if $key.UsedByBackEnd}}
func remove{{$key.Name | ToKeyFuncBaseName}}(data connectionData) {
	removeSettingKey(data, {{$sectionName}}, {{$key.Name}})
}{{end}}{{end}}
`

// general setting utils
const tplGeneralSettingUtils = `// This file is automatically generated, please don't edit manually.
package network

func generalIsKeyInSettingSection(section, key string) bool {
	if isVirtualKey(section, key) {
		return true
	}
	switch section {
	default:
		logger.Warning("invalid section name", section){{range .}}
	case {{.Name}}:
		return isKeyIn{{.Name | ToSectionFuncBaseName}}(key){{end}}
	}
	return false
}

func generalGetSettingKeyType(section, key string) (t ktype) {
	if isVirtualKey(section, key) {
		t = getSettingVkeyType(section, key)
		return
	}
	switch section {
	default:
		logger.Warning("invalid section name", section){{range .}}
	case {{.Name}}:
		t = get{{.Name | ToSectionFuncBaseName}}KeyType(key){{end}}
	}
	return
}

func generalGetSettingAvailableKeys(data connectionData, section string) (keys []string) {
	if isVirtualSection(section) {
		keys = generalGetSettingVsectionAvailableKeys(data, section)
		return
	}
	switch section { {{range .}}
	case {{.Name}}:
		keys = get{{.Name | ToSectionFuncBaseName}}AvailableKeys(data){{end}}
	}
	return
}

func generalGetSettingAvailableValues(data connectionData, section, key string) (values []kvalue) {
	if isVirtualKey(section, key) {
		values = generalGetSettingVkeyAvailableValues(data, section, key)
		return
	}
	switch section { {{range .}}
	case {{.Name}}:
		values = get{{.Name | ToSectionFuncBaseName}}AvailableValues(data, key){{end}}
	}
	return
}

func generalCheckSettingValues(data connectionData, section string) (errs sectionErrors) {
	if isVirtualSection(section) {
		return
	}
	switch section {
	default:
		logger.Error("invalid section name", section){{range .}}
	case {{.Name}}:
		errs = check{{.Name | ToSectionFuncBaseName}}Values(data){{end}}
	}
	return
}

func generalGetSettingKeyJSON(data connectionData, section, key string) (valueJSON string) {
	if isVirtualKey(section, key) {
		valueJSON = generalGetVkeyJSON(data, section, key)
		return
	}
	switch section {
	default:
		logger.Warning("invalid section name", section){{range.}}
	case {{.Name}}:
		valueJSON = generalGet{{.Name | ToSectionFuncBaseName}}KeyJSON(data, key){{end}}
	}
	return
}

func generalSetSettingKeyJSON(data connectionData, section, key, valueJSON string) (err error) {
	if isVirtualKey(section, key) {
		err = generalSetVkeyJSON(data, section, key, valueJSON)
		return
	}
	switch section {
	default:
		logger.Warning("invalid section name", section){{range .}}
	case {{.Name}}:
		err = generalSet{{.Name | ToSectionFuncBaseName}}KeyJSON(data, key, valueJSON){{end}}
	}
	return
}

func generalGetSettingDefaultValue(section, key string) (value interface{}) {
	switch section {
	default:
		logger.Warning("invalid section name", section){{range .}}
	case {{.Name}}:
		value = get{{.Name | ToSectionFuncBaseName}}DefaultValue(key){{end}}
	}
	return
}`

// Virtual key
const tplVkey = `// This file is automatically generated, please don't edit manually.
package network{{$vks := .}}
// Virtual key names
const ({{range .}}
	{{.Name}} = "{{.Value}}"{{end}}
)

// Virtual key data
var virtualKeys = []vkeyInfo{ {{range .}}
	{Value:{{.Name}}, Type:{{.Type}}, VkType:{{.VkType}}, RelatedSection:{{.RelatedSection}}, RelatedKeys:[]string{ {{range $k := .RelatedKeys}}{{$k}},{{end}} }, Available:{{.UsedByFrontEnd}}, Optional:{{.Optional}} },{{end}}
}

// Get JSON value generally
func generalGetVkeyJSON(data connectionData, section, key string) (valueJSON string) {
	switch section { {{range $i, $section := GetAllVkeysRelatedSections $vks}}
	case {{$section}}:
		switch key { {{range $i, $key := GetVkeysOfSection $vks $section}}
		case {{$key}}:
			return get{{$key | ToKeyFuncBaseName}}JSON(data){{end}}
		}{{end}}
	}
	logger.Error("invalid virtual key:", section, key)
	return
}

// Set JSON value generally
func generalSetVkeyJSON(data connectionData, section, key string, valueJSON string) (err error) {
	if isJSONValueMeansToDeleteKey(valueJSON, getSettingVkeyType(section, key)) {
		logger.Debugf("json value means to remove key, data[%s][%s]=%#v", section, key, valueJSON)
		removeVirtualKey(data, section, key)
		return
	}
	// each virtual key own a logic setter
	switch section { {{range $i, $section := GetAllVkeysRelatedSections $vks}}
	case {{$section}}:
		switch key { {{range $i, $key := GetVkeysOfSection $vks $section}}
		case {{$key}}:
			err = logicSet{{$key | ToKeyFuncBaseName}}JSON(data, valueJSON)
			return{{end}}
		}{{end}}
	}
	logger.Error("invalid virtual key:", section, key)
	return
}

// JSON getter{{range $i, $vk := $vks}}{{$keyBaseFuncName := $vk.Name | ToKeyFuncBaseName}}
func get{{$keyBaseFuncName}}JSON(data connectionData) (valueJSON string) {
	valueJSON, _ = marshalJSON(get{{$keyBaseFuncName}}(data))
	return
}{{end}}

// Logic JSON setter{{range $i, $vk := $vks}}{{$keyBaseFuncName := $vk.Name | ToKeyFuncBaseName}}
func logicSet{{$keyBaseFuncName}}JSON(data connectionData, valueJSON string) (err error) {
	value, _ := jsonToKeyValue{{$vk.Type | ToKeyTypeShortName}}(valueJSON)
	return logicSet{{$keyBaseFuncName}}(data, value)
}{{end}}

// Getter for key's enable wrapper{{range $i, $vk := $vks}}{{if IsEnableWrapperVkey $vk.Name}}{{$keyBaseFuncName := $vk.Name | ToKeyFuncBaseName}}
func get{{$keyBaseFuncName}}(data connectionData) (value bool) { {{range $relatedKey := $vk.RelatedKeys}}
	if !is{{$relatedKey | ToKeyFuncBaseName}}Exists(data) {
		return false
	}{{end}}
	return true
}{{end}}{{end}}

// Setter for key's enable wrapper{{range $i, $vk := $vks}}{{if IsEnableWrapperVkey $vk.Name}}{{$keyBaseFuncName := $vk.Name | ToKeyFuncBaseName}}
func logicSet{{$keyBaseFuncName}}(data connectionData, value bool) (err error) {
	if value { {{range $relatedKey := $vk.RelatedKeys}}
		set{{$relatedKey | ToKeyFuncBaseName}}(data, {{$relatedKey | ToKeyDefaultValue}}){{end}}
	} else { {{range $relatedKey := $vk.RelatedKeys}}
		remove{{$relatedKey | ToKeyFuncBaseName}}(data){{end}}
	}
	return
}{{end}}{{end}}

`
