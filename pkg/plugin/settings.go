package plugin

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// Settings - data loaded from grafana settings database
type Settings struct {
	Server                    string `json:"server,omitempty"`
	Port                      int64  `json:"port,omitempty"`
	Username                  string `json:"username,omitempty"`
	DefaultDatabase           string `json:"defaultDatabase,omitempty"`
	InsecureSkipVerify        bool   `json:"tlsSkipVerify,omitempty"`
	TlsClientAuth             bool   `json:"tlsAuth,omitempty"`
	TlsAuthWithCACert         bool   `json:"tlsAuthWithCACert,omitempty"`
	Password                  string `json:"-,omitempty"`
	TlsCACert                 string
	TlsClientCert             string
	TlsClientKey              string
	Secure                    bool            `json:"secure,omitempty"`
	Timezone                  string          `json:"timezone,omitempty"`
	Timeout                   string          `json:"timeout,omitempty"`
	EnableLogsMapFieldFlatten bool            `json:"enableLogsMapFieldFlatten,omitempty"`
	QueryTimeout              string          `json:"queryTimeout,omitempty"`
	CustomSettings            []CustomSetting `json:"customSettings"`
}

type CustomSetting struct {
	Setting string `json:"setting"`
	Value   string `json:"value"`
}

func (settings *Settings) isValid() (err error) {
	if settings.Server == "" {
		return ErrorMessageInvalidServerName
	}
	if settings.Port == 0 {
		return ErrorMessageInvalidPort
	}
	return nil
}

// LoadSettings will read and validate Settings from the DataSourceConfig
func LoadSettings(config backend.DataSourceInstanceSettings) (settings Settings, err error) {
	var jsonData map[string]interface{}
	if err := json.Unmarshal(config.JSONData, &jsonData); err != nil {
		return settings, fmt.Errorf("%s: %w", err.Error(), ErrorMessageInvalidJSON)
	}

	if jsonData["server"] != nil {
		settings.Server = jsonData["server"].(string)
	}
	if jsonData["port"] != nil {
		if port, ok := jsonData["port"].(string); ok {
			settings.Port, err = strconv.ParseInt(port, 0, 64)
			if err != nil {
				return settings, fmt.Errorf("could not parse port value: %w", err)
			}
		} else {
			settings.Port = int64(jsonData["port"].(float64))
		}
	}
	if jsonData["username"] != nil {
		settings.Username = jsonData["username"].(string)
	}
	if jsonData["defaultDatabase"] != nil {
		settings.DefaultDatabase = jsonData["defaultDatabase"].(string)
	}

	if jsonData["tlsSkipVerify"] != nil {
		if tlsSkipVerify, ok := jsonData["tlsSkipVerify"].(string); ok {
			settings.InsecureSkipVerify, err = strconv.ParseBool(tlsSkipVerify)
			if err != nil {
				return settings, fmt.Errorf("could not parse tlsSkipVerify value: %w", err)
			}
		} else {
			settings.InsecureSkipVerify = jsonData["tlsSkipVerify"].(bool)
		}
	}
	if jsonData["tlsAuth"] != nil {
		if tlsAuth, ok := jsonData["tlsAuth"].(string); ok {
			settings.TlsClientAuth, err = strconv.ParseBool(tlsAuth)
			if err != nil {
				return settings, fmt.Errorf("could not parse tlsAuth value: %w", err)
			}
		} else {
			settings.TlsClientAuth = jsonData["tlsAuth"].(bool)
		}
	}
	if jsonData["tlsAuthWithCACert"] != nil {
		if tlsAuthWithCACert, ok := jsonData["tlsAuthWithCACert"].(string); ok {
			settings.TlsAuthWithCACert, err = strconv.ParseBool(tlsAuthWithCACert)
			if err != nil {
				return settings, fmt.Errorf("could not parse tlsAuthWithCACert value: %w", err)
			}
		} else {
			settings.TlsAuthWithCACert = jsonData["tlsAuthWithCACert"].(bool)
		}
	}

	if jsonData["timeout"] != nil {
		settings.Timeout = jsonData["timeout"].(string)
	}
	if jsonData["queryTimeout"] != nil {
		if val, ok := jsonData["queryTimeout"].(string); ok {
			settings.QueryTimeout = val
		}
		if val, ok := jsonData["queryTimeout"].(float64); ok {
			settings.QueryTimeout = fmt.Sprintf("%d", int64(val))
		}
	}
	if jsonData["timezone"] != nil {
		settings.Timezone = jsonData["timezone"].(string)
	}

	if jsonData["enableLogsMapFieldFlatten"] != nil {
		settings.EnableLogsMapFieldFlatten = jsonData["enableLogsMapFieldFlatten"].(bool)
	}

	if jsonData["customSettings"] != nil {
		customSettingsRaw := jsonData["customSettings"].([]interface{})
		customSettings := make([]CustomSetting, len(customSettingsRaw))

		for i, raw := range customSettingsRaw {
			rawMap := raw.(map[string]interface{})
			customSettings[i] = CustomSetting{
				Setting: rawMap["setting"].(string),
				Value:   rawMap["value"].(string),
			}
		}

		settings.CustomSettings = customSettings
	}

	if strings.TrimSpace(settings.Timeout) == "" {
		settings.Timeout = "10"
	}
	if strings.TrimSpace(settings.QueryTimeout) == "" {
		settings.QueryTimeout = "60"
	}
	password, ok := config.DecryptedSecureJSONData["password"]
	if ok {
		settings.Password = password
	}
	tlsCACert, ok := config.DecryptedSecureJSONData["tlsCACert"]
	if ok {
		settings.TlsCACert = tlsCACert
	}
	tlsClientCert, ok := config.DecryptedSecureJSONData["tlsClientCert"]
	if ok {
		settings.TlsClientCert = tlsClientCert
	}
	tlsClientKey, ok := config.DecryptedSecureJSONData["tlsClientKey"]
	if ok {
		settings.TlsClientKey = tlsClientKey
	}

	return settings, settings.isValid()
}
