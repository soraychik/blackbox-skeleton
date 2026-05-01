package models

import "time"

type Device struct {
	ID         int       `json:"id"`
	Hostname   string    `json:"hostname"`
	MgmtIP     *string   `json:"mgmt_ip,omitempty"`
	Vendor     *string   `json:"vendor,omitempty"`
	Model      *string   `json:"model,omitempty"`
	DeviceType string    `json:"device_type,omitempty"`
	SiteCode   string    `json:"site_code,omitempty"`
	Location   string    `json:"location,omitempty"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
}

type ConfigVersion struct {
	ID              int       `json:"id"`
	DeviceID        int       `json:"device_id"`
	VersionHash     string    `json:"version_hash"`
	StorageType     string    `json:"storage_type"`
	StoragePath     string    `json:"storage_path"`
	ParentVersionID *int      `json:"parent_version_id,omitempty"`
	ChainBaseID     *int      `json:"chain_base_id,omitempty"`
	ChainPosition   int       `json:"chain_position"`
	OriginalSize    uint32    `json:"original_size"`
	CompressedSize  uint32    `json:"compressed_size"`
	CreatedAt       time.Time `json:"created_at"`
}

type DiffIndex struct {
	ID              int       `json:"id"`
	LeftVersionID   int       `json:"left_version_id"`
	RightVersionID  int       `json:"right_version_id"`
	AddedLines      int       `json:"added_lines"`
	RemovedLines    int       `json:"removed_lines"`
	DiffStoragePath *string   `json:"diff_storage_path,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type DiffLine struct {
	Type     string `json:"type"`
	Content  string `json:"content"`
	LineNum  int    `json:"line_num"`
	LeftNum  int    `json:"left_num,omitempty"`
	RightNum int    `json:"right_num,omitempty"`
}

type DiffResult struct {
	LeftVersionID  int        `json:"left_version_id"`
	RightVersionID int        `json:"right_version_id"`
	LeftContent    string     `json:"left_content"`
	RightContent   string     `json:"right_content"`
	Lines          []DiffLine `json:"lines"`
}

type SearchChangesRequest struct {
	AddedPatterns   []string `json:"added_patterns"`
	RemovedPatterns []string `json:"removed_patterns"`
	FromDate        string   `json:"from_date"`
	ToDate          string   `json:"to_date"`
}

type SearchCountRequest struct {
	Pattern       string `json:"pattern"`
	CaseSensitive bool   `json:"caseSensitive"`
	Scope         string `json:"scope"`
	DeviceID      *int   `json:"deviceId"`
}

type SearchCountResult struct {
	DeviceID   int           `json:"device_id"`
	VersionID  int           `json:"version_id"`
	Hostname   string        `json:"hostname"`
	MgmtIP     *string       `json:"mgmt_ip,omitempty"`
	MatchCount int           `json:"match_count"`
	Snippets   []SnippetLine `json:"snippets"`
}

type SnippetLine struct {
	Line  int    `json:"line"`
	Text  string `json:"text"`
	Match bool   `json:"match"`
}

type ChangeMatch struct {
	LeftVersionID  int    `json:"left_version_id"`
	RightVersionID int    `json:"right_version_id"`
	LeftDate       string `json:"left_date"`
	RightDate      string `json:"right_date"`
	AddedCount     int    `json:"added_count"`
	RemovedCount   int    `json:"removed_count"`
}

type DeviceChangeResult struct {
	DeviceID   int           `json:"device_id"`
	Hostname   string        `json:"hostname"`
	MgmtIP     *string       `json:"mgmt_ip,omitempty"`
	Vendor     *string       `json:"vendor,omitempty"`
	Model      *string       `json:"model,omitempty"`
	ChangeList []ChangeMatch `json:"changes"`
}

type VersionWithDevice struct {
	ID             int       `json:"id"`
	DeviceID       int       `json:"device_id"`
	DeviceHostname string    `json:"device_hostname"`
	VersionHash    string    `json:"version_hash"`
	StorageType    string    `json:"storage_type"`
	StoragePath    string    `json:"storage_path"`
	ChainPosition  int       `json:"chain_position"`
	OriginalSize   uint32    `json:"original_size"`
	CompressedSize uint32    `json:"compressed_size"`
	CreatedAt      time.Time `json:"created_at"`
}

type TopDevice struct {
	DeviceID    int       `json:"device_id"`
	Hostname    string    `json:"hostname"`
	ChangeCount int       `json:"change_count"`
	LastChange  time.Time `json:"last_change"`
}

type VersionPair struct {
	Left  *ConfigVersion
	Right *ConfigVersion
}

type DeviceRow struct {
	ID       int
	Hostname string
	MgmtIP   *string
	Vendor   *string
	Model    *string
}

type DeviceVersionRow struct {
	DeviceID    int
	Hostname    string
	MgmtIP      *string
	VersionID   int
	StorageType string
	StoragePath string
}

type SystemSettings struct {
	ConfigSourceType    string  `json:"config_source_type"`
	ConfigSourcePath    string  `json:"config_source_path"`
	SmbUsername         string  `json:"smb_username,omitempty"`
	SmbDomain           string  `json:"smb_domain,omitempty"`
	ScanIntervalSeconds int     `json:"scan_interval_seconds"`
	DiffThreshold       float64 `json:"diff_threshold"`
	// SmbPassword намеренно не возвращается в ответе API

	LDAPEnabled      bool   `json:"ldap_enabled"`
	LDAPUrl          string `json:"ldap_url,omitempty"`
	LDAPBindDN       string `json:"ldap_bind_dn,omitempty"`
	LDAPUserBase     string `json:"ldap_user_base,omitempty"`
	LDAPUserFilter   string `json:"ldap_user_filter,omitempty"`
	LDAPRoleAdmin    string `json:"ldap_role_admin,omitempty"`
	LDAPRoleEngineer string `json:"ldap_role_engineer,omitempty"`
	LDAPRoleOperator string `json:"ldap_role_operator,omitempty"`
	// LDAPBindPassword намеренно не возвращается в ответе API
}

type SettingsUpdateRequest struct {
	ConfigSourceType    string  `json:"config_source_type"`
	ConfigSourcePath    string  `json:"config_source_path"`
	SmbUsername         string  `json:"smb_username"`
	SmbPassword         string  `json:"smb_password"`
	SmbDomain           string  `json:"smb_domain"`
	ScanIntervalSeconds int     `json:"scan_interval_seconds"`
	DiffThreshold       float64 `json:"diff_threshold"`

	LDAPEnabled      bool   `json:"ldap_enabled"`
	LDAPUrl          string `json:"ldap_url"`
	LDAPBindDN       string `json:"ldap_bind_dn"`
	LDAPBindPassword string `json:"ldap_bind_password"`
	LDAPUserBase     string `json:"ldap_user_base"`
	LDAPUserFilter   string `json:"ldap_user_filter"`
	LDAPRoleAdmin    string `json:"ldap_role_admin"`
	LDAPRoleEngineer string `json:"ldap_role_engineer"`
	LDAPRoleOperator string `json:"ldap_role_operator"`
}