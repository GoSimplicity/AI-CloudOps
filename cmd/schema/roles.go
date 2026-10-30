package main

import (
	"errors"
	"fmt"
	"strings"
)

var errRolePlan = errors.New("invalid schema role plan identifiers")

// RoleAccount identifies an existing MySQL role. Both fields must be explicitly
// supplied: a migration tool must not guess a user name or account host.
// This is a role reference, not a password-bearing application account.
type RoleAccount struct {
	Name string
	Host string
}

// RolePlan contains offline GRANT statements only. It does not create roles,
// users, credentials or connections, and does not revoke existing privileges.
// Apply Migrator before initialization; apply Runtime only after tables exist.
// A dedicated administrator must provision the roles and review all inherited
// privileges before applying this plan to a separately provisioned database.
type RolePlan struct {
	Runtime  []string
	Migrator []string
}

const (
	schemaHistoryTable     = "cloudops_schema_history"
	notificationClockTable = "cl_workorder_notification_clock"
)

// baselineBusinessTables is the reviewed version-1 inventory, including the
// clock table. Do not derive privileges by parsing arbitrary SQL at runtime.
// Tests compare this fixed inventory exactly with the immutable baseline.
var baselineBusinessTables = [...]string{
	"cl_cron_jobs",
	"cl_k8s_clusters",
	"cl_k8s_yaml_task",
	"cl_k8s_yaml_template",
	"cl_monitor_alert_events",
	"cl_monitor_alert_manager_pools",
	"cl_monitor_alert_rules",
	"cl_monitor_configs",
	"cl_monitor_on_duty_changes",
	"cl_monitor_on_duty_group_users",
	"cl_monitor_on_duty_groups",
	"cl_monitor_on_duty_histories",
	"cl_monitor_record_rules",
	"cl_monitor_scrape_jobs",
	"cl_monitor_scrape_pools",
	"cl_monitor_send_group_first_upgrade_users",
	"cl_monitor_send_group_second_upgrade_users",
	"cl_monitor_send_group_static_receive_users",
	"cl_monitor_send_groups",
	"cl_system_apis",
	"cl_system_audit_logs",
	"cl_system_departments",
	"cl_system_role_apis",
	"cl_system_roles",
	"cl_system_user_roles",
	"cl_system_users",
	"cl_tree_cloud_account",
	"cl_tree_cloud_account_region",
	"cl_tree_cloud_managed_operation",
	"cl_tree_cloud_managed_resource",
	"cl_tree_cloud_managed_sync_history",
	"cl_tree_cloud_resource",
	"cl_tree_cloud_resource_change_log",
	"cl_tree_cloud_resource_sync_history",
	"cl_tree_cloud_sync_lease",
	"cl_tree_local_resource",
	"cl_tree_node",
	"cl_tree_node_admin",
	"cl_tree_node_cloud",
	"cl_tree_node_local",
	"cl_tree_node_member",
	"cl_user_apis",
	"cl_workorder_category",
	"cl_workorder_form_design",
	"cl_workorder_inbox_message",
	"cl_workorder_instance",
	"cl_workorder_instance_comment",
	"cl_workorder_instance_flow",
	"cl_workorder_instance_timeline",
	"cl_workorder_notification",
	notificationClockTable,
	"cl_workorder_notification_delivery",
	"cl_workorder_notification_event",
	"cl_workorder_notification_log",
	"cl_workorder_notification_queue",
	"cl_workorder_notification_resolution",
	"cl_workorder_process",
	"cl_workorder_template",
}

// BuildRolePlan produces grants for the immutable version-1 schema. Migrator
// DDL and DML are table-scoped; schema REFERENCES permits complete metadata
// inspection without granting unknown tables row access. Applying
// these grants cannot remove older broad or inherited privileges from a role.
func BuildRolePlan(database string, runtimeRole, migratorRole RoleAccount) (RolePlan, error) {
	if !validDatabase(database) || !validRole(runtimeRole) || !validRole(migratorRole) ||
		(runtimeRole.Name == migratorRole.Name && strings.EqualFold(runtimeRole.Host, migratorRole.Host)) {
		return RolePlan{}, errRolePlan
	}
	accountSQL := func(role RoleAccount) string { return "'" + role.Name + "'@'" + role.Host + "'" }
	runtimeAccount, migratorAccount := accountSQL(runtimeRole), accountSQL(migratorRole)
	target := func(table string) string { return "`" + database + "`.`" + table + "`" }
	grant := func(privileges, table, account string) string {
		return fmt.Sprintf("GRANT %s ON %s TO %s;", privileges, target(table), account)
	}
	// REFERENCES exposes complete schema metadata, including unknown legacy
	// tables, without granting their row contents or mutation. Database-level
	// GRANT patterns treat '_' and '%' specially on default MySQL settings;
	// validDatabase therefore permits ASCII letters and digits only.
	plan := RolePlan{Migrator: []string{fmt.Sprintf("GRANT REFERENCES ON `%s`.* TO %s;", database, migratorAccount)}}
	for _, table := range baselineBusinessTables {
		runtimePrivileges := "SELECT, INSERT, UPDATE, DELETE"
		if table == notificationClockTable {
			runtimePrivileges = "SELECT"
		}
		plan.Runtime = append(plan.Runtime, grant(runtimePrivileges, table, runtimeAccount))
		migratorPrivileges := "SELECT, CREATE, ALTER, INDEX"
		switch table {
		case "cl_tree_cloud_resource_change_log", "cl_workorder_notification":
			migratorPrivileges += ", UPDATE"
		case notificationClockTable:
			migratorPrivileges += ", INSERT, UPDATE"
		}
		plan.Migrator = append(plan.Migrator, grant(migratorPrivileges, table, migratorAccount))
	}
	plan.Runtime = append(plan.Runtime, grant("SELECT", schemaHistoryTable, runtimeAccount))
	// Only the two frozen data repairs need UPDATE on business tables. Clock
	// initialization inserts a marker; MySQL requires UPDATE for its FOR UPDATE
	// locking reads. The command never rewrites a marker, although this trusted
	// migration role can do so outside the command.
	plan.Migrator = append(plan.Migrator,
		grant("SELECT, INSERT, UPDATE, DELETE, CREATE, ALTER, INDEX", schemaHistoryTable, migratorAccount),
	)
	return plan, nil
}

func validIdentifier(value string, maximum int) bool {
	if value == "" || len(value) > maximum {
		return false
	}
	for _, char := range value {
		if char != '_' && (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}

func validDatabase(database string) bool {
	if !validIdentifier(database, 64) || strings.Contains(database, "_") {
		return false
	}
	switch strings.ToLower(database) {
	case "mysql", "sys", "performance_schema", "information_schema":
		return false
	default:
		return true
	}
}

func validRole(role RoleAccount) bool {
	if !validIdentifier(role.Name, 32) || role.Host == "" || len(role.Host) > 255 {
		return false
	}
	for _, char := range role.Host {
		if char != '.' && char != ':' && char != '-' && char != '%' && char != '_' &&
			(char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}
