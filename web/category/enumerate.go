package category

// ChaosbladeType Name
type ChaosbladeType string

const (
	ChaosbladeTypeUnknown ChaosbladeType = ""       // Unknown type
	ChaosbladeTypeCPU     ChaosbladeType = "cpu"    // CPU
	ChaosbladeTypeIO      ChaosbladeType = "IO"     // IO
	ChaosbladeTypeMemory  ChaosbladeType = "mem"    // memory
	ChaosbladeTypeScript  ChaosbladeType = "script" // script
)

type ChaosbladeCPUType string

const (
	ChaosbladeCPUTypeUnknown  ChaosbladeCPUType = ""         // Unknown type
	ChaosbladeTypeCPUFullLoad ChaosbladeCPUType = "fullload" // CPU Fullload
)

type ChaosbladeMemoryType string

const (
	ChaosbladeMemoryTypeUnknown ChaosbladeMemoryType = ""     // Unknown type
	ChaosbladeMemoryTypeLoad    ChaosbladeMemoryType = "load" // Memory load
)

type ChaosbladeScriptType string

const (
	ChaosbladeScriptTypeUnknown ChaosbladeScriptType = ""        // Unknown type
	ChaosbladeScriptTypeDelay   ChaosbladeScriptType = "delay"   // script delay
	ChaosbladeScriptTypeExecute ChaosbladeScriptType = "execute" // script execute
	ChaosbladeScriptTypeExit    ChaosbladeScriptType = "Exit"    // script Exit
)
