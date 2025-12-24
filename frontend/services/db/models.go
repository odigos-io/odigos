package db

import (
	"time"

	"gorm.io/gorm"
)

// InstrumentedProcess represents a process instrumented inside a Kubernetes pod
//
//	**Primary Key Fields:**
//
// - `process_pid` → The process ID inside the container (not unique by itself).
// - `k8s_pod_name` → The pod in which the process is running (pods can have multiple containers).
// - `k8s_namespace_name` → Ensures uniqueness across namespaces (pods can have the same name in different namespaces).
// - `k8s_container_name` → A pod may contain multiple containers, each with its own PID space.
// - `created_at` → Ensures uniqueness even if the same PID is reused after a pod restart.

type InstrumentedProcess struct {
	OdigletName       string    `gorm:"column:odiglet_name; type:text; not null"`                                                                                                    // Agent/collector name
	NodeName          string    `gorm:"column:k8s_node_name; type:text; not null"`                                                                                                   // Kubernetes node
	WorkloadName      string    `gorm:"column:workload_name; type:text; not null"`                                                                                                   // Kubernetes workload name
	WorkloadKind      string    `gorm:"column:workload_kind; type:text; not null; check:workload_kind IN ('Deployment', 'StatefulSet', 'DaemonSet', 'CronJob', 'DeploymentConfig', 'Rollout')"` // Kubernetes workload kind [Deployment/StatefulSet/DaemonSet/CronJob/DeploymentConfig/Rollout]
	PodName           string    `gorm:"primaryKey; column:k8s_pod_name; type:text; not null"`                                                                                        // Kubernetes pod name (PK)
	PID               int       `gorm:"primaryKey; column:process_pid; not null"`                                                                                                    // Process ID inside the container (PK)
	Namespace         string    `gorm:"primaryKey; column:k8s_namespace_name; type:text; not null"`                                                                                  // Kubernetes namespace (PK)
	ContainerName     string    `gorm:"primaryKey; column:k8s_container_name; type:text; not null"`                                                                                  // Container name inside the pod (PK)
	CreatedAt         time.Time `gorm:"primaryKey; column:created_at; not null; default:CURRENT_TIMESTAMP"`                                                                          // Timestamp when the process was detected (PK)
	TelemetryLang     string    `gorm:"column:telemetry_sdk_language; type:text; not null"`                                                                                          // Language of the telemetry SDK
	ServiceInstanceID string    `gorm:"column:service_instance_id; type:text; not null"`                                                                                             // Service instance identifier
	Healthy           bool      `gorm:"column:healthy; not null"`                                                                                                                    // Process health status
	HealthyReason     string    `gorm:"column:healthy_reason; type:text"`                                                                                                            // Reason for health status (optional)
}

func (InstrumentedProcess) TableName() string {
	return "instrumented_processes"
}

// InstrumentedProcessError stores processes that are marked as unhealthy (`healthy = false`)
// along with their `healthy_reason`. This table automatically gets updated via a database
// trigger whenever a new healthy process is inserted into `InstrumentedProcesses`.
// It also has a foreign key constraint with `OnDelete:CASCADE`, ensuring that
// errors are automatically removed when the corresponding process is deleted.

type InstrumentedProcessError struct {
	PID           int       `gorm:"primaryKey; column:process_pid; not null"`
	PodName       string    `gorm:"primaryKey; column:k8s_pod_name; type:text; not null"`
	Namespace     string    `gorm:"primaryKey; column:k8s_namespace_name; type:text; not null"`
	ContainerName string    `gorm:"primaryKey; column:k8s_container_name; type:text; not null"`
	CreatedAt     time.Time `gorm:"primaryKey; column:created_at; not null; default:CURRENT_TIMESTAMP"`
	HealthyReason string    `gorm:"column:healthy_reason; type:text; not null"`

	// Foreign Key Reference (Auto-delete errors when the process is removed)
	InstrumentedProcess InstrumentedProcess `gorm:"constraint:OnDelete:CASCADE;foreignKey:PID,PodName,Namespace,ContainerName,CreatedAt"`
}

func (InstrumentedProcessError) TableName() string {
	return "instrumented_processes_errors"
}

func InitializeDatabaseSchema(db *gorm.DB) {
	db.AutoMigrate(&InstrumentedProcess{})
	db.AutoMigrate(&InstrumentedProcessError{})

	// Drop existing triggers if they exist
	db.Exec(`DROP TRIGGER IF EXISTS delete_process_errors;`)
	db.Exec(`DROP TRIGGER IF EXISTS insert_process_errors;`)

	// Triggers to automatically manage the `instrumented_processes_errors` table.
	// - `insert_process_errors`: Inserts or updates an entry in `instrumented_processes_errors` when a process becomes unhealthy (`healthy = false`).
	// - `delete_process_errors`: Removes the corresponding error when a process transitions from unhealthy to healthy (`healthy = false → true`).
	// These triggers ensure that the error table always reflects the current state of unhealthy processes without requiring manual inserts or deletes.
	db.Exec(`
    CREATE TRIGGER IF NOT EXISTS insert_process_errors
    AFTER INSERT ON instrumented_processes
    WHEN NEW.healthy = 0
    BEGIN
        INSERT INTO instrumented_processes_errors 
            (process_pid, k8s_pod_name, k8s_namespace_name, k8s_container_name, created_at, healthy_reason)
        VALUES 
            (NEW.process_pid, NEW.k8s_pod_name, NEW.k8s_namespace_name, NEW.k8s_container_name, NEW.created_at, NEW.healthy_reason)
        ON CONFLICT (process_pid, k8s_pod_name, k8s_namespace_name, k8s_container_name, created_at) 
        DO UPDATE SET healthy_reason = NEW.healthy_reason;
    END;
`)

	db.Exec(`
    CREATE TRIGGER IF NOT EXISTS delete_process_errors
    AFTER UPDATE ON instrumented_processes
    WHEN OLD.healthy = 0 AND NEW.healthy = 1
    BEGIN
        DELETE FROM instrumented_processes_errors
        WHERE process_pid = NEW.process_pid
        AND k8s_pod_name = NEW.k8s_pod_name
        AND k8s_namespace_name = NEW.k8s_namespace_name
        AND k8s_container_name = NEW.k8s_container_name
        AND created_at = OLD.created_at;
    END;
`)

}
