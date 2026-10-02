package permissions

const (
	SystemAdmin  = "system.admin"
	SystemManage = "system.manage"

	UsersRead   = "users.read"
	UsersWrite  = "users.write"
	UsersDelete = "users.delete"

	ReportsView   = "reports.view"
	ReportsExport = "reports.export"

	MessagesRead    = "messages.read"
	MessagesWrite   = "messages.write"
	MessagingAccess = "messaging.access"

	DocumentsRead  = "documents.read"
	DocumentsWrite = "documents.write"
)

// Default permission sets
var (
	// DefaultCitizenPermissions is the baseline permission set assigned to typical citizens.
	DefaultCitizenPermissions = []string{
		DocumentsRead,
		DocumentsWrite,
		"identity.me.read",
		"notifications.read",
		"sessions.manage",
	}

	// DefaultOfficerPermissions extend citizen permissions with internal staff capabilities.
	DefaultOfficerPermissions = append(DefaultCitizenPermissions, []string{
		UsersRead,
		UsersWrite,
		"cases.manage",
		"officer.actions",
	}...)

	// DefaultAdminPermissions grants broad system-level rights.
	DefaultAdminPermissions = append(DefaultOfficerPermissions, []string{
		SystemAdmin,
		SystemManage,
		ReportsView,
		ReportsExport,
	}...)
)
