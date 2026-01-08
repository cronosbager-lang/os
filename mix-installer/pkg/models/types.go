package models

// InstallConfig holds the complete installation configuration
type InstallConfig struct {
	Mode       InstallMode       `json:"mode"`
	Disk       DiskConfig        `json:"disk"`
	Partitions []PartitionConfig `json:"partitions"`
	User       UserConfig        `json:"user"`
	System     SystemConfig      `json:"system"`
	Packages   PackageConfig     `json:"packages"`
	Bootloader BootloaderConfig  `json:"bootloader"`
}

// InstallMode represents the installation mode
type InstallMode string

const (
	ModeAutonomous InstallMode = "autonomous"
	ModeGuided     InstallMode = "guided"
	ModeManual     InstallMode = "manual"
)

// DiskConfig holds disk configuration
type DiskConfig struct {
	Device     string `json:"device"`
	Model      string `json:"model"`
	Size       uint64 `json:"size"`
	Type       string `json:"type"` // sata, nvme, usb
	WipeAll    bool   `json:"wipe_all"`
	UseEntire  bool   `json:"use_entire"`
	TableType  string `json:"table_type"` // gpt, mbr
}

// PartitionConfig holds partition configuration
type PartitionConfig struct {
	Number     int    `json:"number"`
	Device     string `json:"device"`
	MountPoint string `json:"mount_point"`
	Size       string `json:"size"` // e.g., "512M", "50G", "100%"
	SizeBytes  uint64 `json:"size_bytes"`
	FSType     string `json:"fs_type"`
	Label      string `json:"label"`
	Flags      []string `json:"flags"` // esp, boot, swap
	Format     bool   `json:"format"`
}

// UserConfig holds user configuration
type UserConfig struct {
	Username       string   `json:"username"`
	Password       string   `json:"password"`
	FullName       string   `json:"full_name"`
	Groups         []string `json:"groups"`
	Shell          string   `json:"shell"`
	AutoLogin      bool     `json:"auto_login"`
	RootPassword   string   `json:"root_password"`
	EnableRootLogin bool    `json:"enable_root_login"`
}

// SystemConfig holds system configuration
type SystemConfig struct {
	Hostname   string `json:"hostname"`
	Timezone   string `json:"timezone"`
	Locale     string `json:"locale"`
	Keymap     string `json:"keymap"`
	Language   string `json:"language"`
}

// PackageConfig holds package selection
type PackageConfig struct {
	Profile    string   `json:"profile"` // minimal, developer, full
	Base       []string `json:"base"`
	Additional []string `json:"additional"`
	Exclude    []string `json:"exclude"`
}

// BootloaderConfig holds bootloader configuration
type BootloaderConfig struct {
	Type       string `json:"type"` // grub, systemd-boot
	Target     string `json:"target"` // x86_64-efi, i386-pc
	Device     string `json:"device"`
	EFIDir     string `json:"efi_dir"`
	BootloaderID string `json:"bootloader_id"`
}

// DiskInfo represents detected disk information
type DiskInfo struct {
	Device     string          `json:"device"`
	Model      string          `json:"model"`
	Serial     string          `json:"serial"`
	Size       uint64          `json:"size"`
	SizeHuman  string          `json:"size_human"`
	Type       string          `json:"type"`
	Transport  string          `json:"transport"`
	Removable  bool            `json:"removable"`
	ReadOnly   bool            `json:"read_only"`
	Partitions []PartitionInfo `json:"partitions"`
}

// PartitionInfo represents detected partition information
type PartitionInfo struct {
	Device     string `json:"device"`
	Number     int    `json:"number"`
	Start      uint64 `json:"start"`
	End        uint64 `json:"end"`
	Size       uint64 `json:"size"`
	SizeHuman  string `json:"size_human"`
	FSType     string `json:"fs_type"`
	Label      string `json:"label"`
	UUID       string `json:"uuid"`
	MountPoint string `json:"mount_point"`
	Flags      []string `json:"flags"`
}

// HardwareInfo represents detected hardware
type HardwareInfo struct {
	CPU       CPUInfo       `json:"cpu"`
	Memory    MemoryInfo    `json:"memory"`
	Disks     []DiskInfo    `json:"disks"`
	Network   []NetworkInfo `json:"network"`
	GPU       []GPUInfo     `json:"gpu"`
	BootMode  string        `json:"boot_mode"` // uefi, legacy
}

// CPUInfo represents CPU information
type CPUInfo struct {
	Model    string `json:"model"`
	Vendor   string `json:"vendor"`
	Cores    int    `json:"cores"`
	Threads  int    `json:"threads"`
	Arch     string `json:"arch"`
	Features []string `json:"features"`
}

// MemoryInfo represents memory information
type MemoryInfo struct {
	Total     uint64 `json:"total"`
	Available uint64 `json:"available"`
	SwapTotal uint64 `json:"swap_total"`
}

// NetworkInfo represents network interface information
type NetworkInfo struct {
	Name       string `json:"name"`
	MAC        string `json:"mac"`
	Driver     string `json:"driver"`
	Connected  bool   `json:"connected"`
	Speed      int    `json:"speed"`
}

// GPUInfo represents GPU information
type GPUInfo struct {
	Name   string `json:"name"`
	Vendor string `json:"vendor"`
	Driver string `json:"driver"`
}

// InstallProgress represents installation progress
type InstallProgress struct {
	Phase       string  `json:"phase"`
	Step        string  `json:"step"`
	Progress    float64 `json:"progress"` // 0-100
	Message     string  `json:"message"`
	Error       string  `json:"error,omitempty"`
	StartedAt   int64   `json:"started_at"`
	CompletedAt int64   `json:"completed_at,omitempty"`
}

// InstallResult represents the installation result
type InstallResult struct {
	Success     bool     `json:"success"`
	Message     string   `json:"message"`
	Errors      []string `json:"errors,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
	Duration    int64    `json:"duration"` // seconds
	RebootReady bool     `json:"reboot_ready"`
}

// AIRecommendation represents an AI-generated recommendation
type AIRecommendation struct {
	Type        string                 `json:"type"`
	Confidence  float64                `json:"confidence"`
	Reason      string                 `json:"reason"`
	Value       interface{}            `json:"value"`
	Alternatives []interface{}         `json:"alternatives,omitempty"`
}

// InstallPlan represents the AI-generated installation plan
type InstallPlan struct {
	Config          InstallConfig      `json:"config"`
	Recommendations []AIRecommendation `json:"recommendations"`
	EstimatedTime   int                `json:"estimated_time"` // minutes
	RiskLevel       string             `json:"risk_level"`     // low, medium, high
	Warnings        []string           `json:"warnings"`
}
