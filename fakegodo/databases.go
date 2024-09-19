package fakegodo

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/digitalocean/godo"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/util/rand"
)

const (
	// CreatingStatus is the status when a database is created.
	CreatingStatus = "creating"
	// OnlineStatus is the status when a database is fetched after creation.
	OnlineStatus = "online"
)

var (
	okResponse       = &godo.Response{Response: &http.Response{StatusCode: http.StatusOK}}
	notFoundResponse = &godo.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}
)

// FakeDatabasesService is a fake godo DatabasesService with limited functionality:
// * Create creates and returns a new database object with the status "creating".
// * Get returns a previously-created database object with the status "online".
// * Delete deletes a previously-created database object.
// * Resize updates a previously-created object with the provided parameters.
type FakeDatabasesService struct {
	Options *godo.DatabaseOptions

	mu        sync.RWMutex
	databases []godo.Database
	users     map[string][]godo.DatabaseUser
}

// UpdatePool ...
func (f *FakeDatabasesService) UpdatePool(_ context.Context, _ string, _ string, _ *godo.DatabaseUpdatePoolRequest) (*godo.Response, error) {
	panic("implement me")
}

// PromoteReplicaToPrimary ...
func (f *FakeDatabasesService) PromoteReplicaToPrimary(_ context.Context, _ string, _ string) (*godo.Response, error) {
	panic("implement me")
}

// UpgradeMajorVersion ...
func (f *FakeDatabasesService) UpgradeMajorVersion(_ context.Context, _ string, _ *godo.UpgradeVersionRequest) (*godo.Response, error) {
	panic("implement me")
}

// List ...
func (f *FakeDatabasesService) List(_ context.Context, _ *godo.ListOptions) ([]godo.Database, *godo.Response, error) {
	panic("not implemented")
}

// Get ...
func (f *FakeDatabasesService) Get(_ context.Context, dbUUID string) (*godo.Database, *godo.Response, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for i := range f.databases {
		db := &f.databases[i]
		if db.ID == dbUUID {
			db.Status = OnlineStatus
			cpy := *db
			return &cpy, okResponse, nil
		}
	}

	return nil, notFoundResponse, errors.New("not found")
}

// GetCA ...
func (f *FakeDatabasesService) GetCA(_ context.Context, _ string) (*godo.DatabaseCA, *godo.Response, error) {
	ca := godo.DatabaseCA{
		Certificate: []byte{01, 02, 03, 04, 05},
	}
	return &ca, okResponse, nil
}

// Create ...
func (f *FakeDatabasesService) Create(_ context.Context, req *godo.DatabaseCreateRequest) (*godo.Database, *godo.Response, error) {
	db := godo.Database{
		ID:          uuid.New().String(),
		Name:        req.Name,
		EngineSlug:  req.EngineSlug,
		VersionSlug: req.Version,
		NumNodes:    req.NumNodes,
		SizeSlug:    req.SizeSlug,
		RegionSlug:  req.Region,
		CreatedAt:   time.Now(),
		Connection: &godo.DatabaseConnection{
			URI:      "uri",
			Database: "database",
			Host:     "host",
			Port:     12345,
			User:     "user",
			Password: "password",
			SSL:      true,
		},
		PrivateConnection: &godo.DatabaseConnection{
			URI:      "private-uri",
			Database: "private-database",
			Host:     "private_host",
			Port:     12345,
			User:     "private-user",
			Password: "private-password",
			SSL:      true,
		},
		Status: CreatingStatus,
	}

	f.mu.Lock()
	f.databases = append(f.databases, db)
	f.mu.Unlock()

	return &db, okResponse, nil
}

// Delete ...
func (f *FakeDatabasesService) Delete(_ context.Context, dbUUID string) (*godo.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.databases {
		db := &f.databases[i]
		if db.ID == dbUUID {
			f.databases = append(f.databases[:i], f.databases[i+1:]...)
			return okResponse, nil
		}
	}

	return notFoundResponse, errors.New("not found")
}

// Resize ...
func (f *FakeDatabasesService) Resize(_ context.Context, dbUUID string, req *godo.DatabaseResizeRequest) (*godo.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.databases {
		db := &f.databases[i]
		if db.ID == dbUUID {
			db.NumNodes = req.NumNodes
			db.SizeSlug = req.SizeSlug
			return okResponse, nil
		}
	}

	return notFoundResponse, errors.New("not found")
}

// Migrate ...
func (f *FakeDatabasesService) Migrate(_ context.Context, _ string, _ *godo.DatabaseMigrateRequest) (*godo.Response, error) {
	panic("not implemented")
}

// UpdateMaintenance ...
func (f *FakeDatabasesService) UpdateMaintenance(_ context.Context, _ string, _ *godo.DatabaseUpdateMaintenanceRequest) (*godo.Response, error) {
	panic("not implemented")
}

// ListBackups ...
func (f *FakeDatabasesService) ListBackups(_ context.Context, _ string, _ *godo.ListOptions) ([]godo.DatabaseBackup, *godo.Response, error) {
	panic("not implemented")
}

// GetUser ...
func (f *FakeDatabasesService) GetUser(_ context.Context, dbUUID string, username string) (*godo.DatabaseUser, *godo.Response, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, db := range f.databases {
		if db.ID == dbUUID {
			for i := range f.users[dbUUID] {
				user := &f.users[dbUUID][i]
				if user.Name == username {
					cpy := *user
					return &cpy, okResponse, nil
				}
			}
		}
	}

	return nil, notFoundResponse, errors.New("not found")
}

// ListUsers ...
func (f *FakeDatabasesService) ListUsers(_ context.Context, _ string, _ *godo.ListOptions) ([]godo.DatabaseUser, *godo.Response, error) {
	panic("not implemented")
}

// CreateUser ...
func (f *FakeDatabasesService) CreateUser(_ context.Context, dbUUID string, req *godo.DatabaseCreateUserRequest) (*godo.DatabaseUser, *godo.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.users == nil {
		f.users = make(map[string][]godo.DatabaseUser)
	}

	for _, db := range f.databases {
		if db.ID == dbUUID {
			if db.Status == CreatingStatus {
				return nil, &godo.Response{Response: &http.Response{StatusCode: http.StatusBadRequest}}, errors.New("db not ready")
			}

			u := godo.DatabaseUser{
				Name:     req.Name,
				Role:     "normal",
				Password: rand.String(16),
			}
			f.users[dbUUID] = append(f.users[dbUUID], u)
			return &u, okResponse, nil
		}
	}

	return nil, notFoundResponse, errors.New("not found")
}

// UpdateUser ...
func (f *FakeDatabasesService) UpdateUser(context.Context, string, string, *godo.DatabaseUpdateUserRequest) (*godo.DatabaseUser, *godo.Response, error) {
	panic("not implemented")
}

// DeleteUser ...
func (f *FakeDatabasesService) DeleteUser(_ context.Context, dbUUID string, username string) (*godo.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, db := range f.databases {
		if db.ID == dbUUID {
			for i := range f.users[dbUUID] {
				u := &f.users[dbUUID][i]
				if u.Name == username {
					f.users[dbUUID] = append(f.users[dbUUID][:i], f.users[dbUUID][i+1:]...)
					return okResponse, nil
				}
			}
		}
	}

	return notFoundResponse, errors.New("not found")
}

// ResetUserAuth ...
func (f *FakeDatabasesService) ResetUserAuth(_ context.Context, _ string, _ string, _ *godo.DatabaseResetUserAuthRequest) (*godo.DatabaseUser, *godo.Response, error) {
	panic("not implemented")
}

// ListDBs ...
func (f *FakeDatabasesService) ListDBs(_ context.Context, _ string, _ *godo.ListOptions) ([]godo.DatabaseDB, *godo.Response, error) {
	panic("not implemented")
}

// CreateDB ...
func (f *FakeDatabasesService) CreateDB(_ context.Context, _ string, _ *godo.DatabaseCreateDBRequest) (*godo.DatabaseDB, *godo.Response, error) {
	panic("not implemented")
}

// GetDB ...
func (f *FakeDatabasesService) GetDB(_ context.Context, _ string, _ string) (*godo.DatabaseDB, *godo.Response, error) {
	panic("not implemented")
}

// DeleteDB ...
func (f *FakeDatabasesService) DeleteDB(_ context.Context, _ string, _ string) (*godo.Response, error) {
	panic("not implemented")
}

// ListPools ...
func (f *FakeDatabasesService) ListPools(_ context.Context, _ string, _ *godo.ListOptions) ([]godo.DatabasePool, *godo.Response, error) {
	panic("not implemented")
}

// CreatePool ...
func (f *FakeDatabasesService) CreatePool(_ context.Context, _ string, _ *godo.DatabaseCreatePoolRequest) (*godo.DatabasePool, *godo.Response, error) {
	panic("not implemented")
}

// GetPool ...
func (f *FakeDatabasesService) GetPool(_ context.Context, _ string, _ string) (*godo.DatabasePool, *godo.Response, error) {
	panic("not implemented")
}

// DeletePool ...
func (f *FakeDatabasesService) DeletePool(_ context.Context, _ string, _ string) (*godo.Response, error) {
	panic("not implemented")
}

// GetReplica ...
func (f *FakeDatabasesService) GetReplica(_ context.Context, _ string, _ string) (*godo.DatabaseReplica, *godo.Response, error) {
	panic("not implemented")
}

// ListReplicas ...
func (f *FakeDatabasesService) ListReplicas(_ context.Context, _ string, _ *godo.ListOptions) ([]godo.DatabaseReplica, *godo.Response, error) {
	panic("not implemented")
}

// CreateReplica ...
func (f *FakeDatabasesService) CreateReplica(_ context.Context, _ string, _ *godo.DatabaseCreateReplicaRequest) (*godo.DatabaseReplica, *godo.Response, error) {
	panic("not implemented")
}

// DeleteReplica ...
func (f *FakeDatabasesService) DeleteReplica(_ context.Context, _ string, _ string) (*godo.Response, error) {
	panic("not implemented")
}

// GetEvictionPolicy ...
func (f *FakeDatabasesService) GetEvictionPolicy(_ context.Context, _ string) (string, *godo.Response, error) {
	panic("not implemented")
}

// SetEvictionPolicy ...
func (f *FakeDatabasesService) SetEvictionPolicy(_ context.Context, _ string, _ string) (*godo.Response, error) {
	panic("not implemented")
}

// GetSQLMode ...
func (f *FakeDatabasesService) GetSQLMode(_ context.Context, _ string) (string, *godo.Response, error) {
	panic("not implemented")
}

// SetSQLMode ...
func (f *FakeDatabasesService) SetSQLMode(_ context.Context, _ string, _ ...string) (*godo.Response, error) {
	panic("not implemented")
}

// GetFirewallRules ...
func (f *FakeDatabasesService) GetFirewallRules(_ context.Context, _ string) ([]godo.DatabaseFirewallRule, *godo.Response, error) {
	panic("not implemented")
}

// UpdateFirewallRules ...
func (f *FakeDatabasesService) UpdateFirewallRules(_ context.Context, _ string, _ *godo.DatabaseUpdateFirewallRulesRequest) (*godo.Response, error) {
	panic("not implemented")
}

// GetPostgreSQLConfig ...
func (f *FakeDatabasesService) GetPostgreSQLConfig(_ context.Context, _ string) (*godo.PostgreSQLConfig, *godo.Response, error) {
	panic("not implemented")
}

// GetRedisConfig ...
func (f *FakeDatabasesService) GetRedisConfig(_ context.Context, _ string) (*godo.RedisConfig, *godo.Response, error) {
	panic("not implemented")
}

// GetMySQLConfig ...
func (f *FakeDatabasesService) GetMySQLConfig(_ context.Context, _ string) (*godo.MySQLConfig, *godo.Response, error) {
	panic("not implemented")
}

// UpdatePostgreSQLConfig ...
func (f *FakeDatabasesService) UpdatePostgreSQLConfig(_ context.Context, _ string, _ *godo.PostgreSQLConfig) (*godo.Response, error) {
	panic("not implemented")
}

// UpdateRedisConfig ...
func (f *FakeDatabasesService) UpdateRedisConfig(_ context.Context, _ string, _ *godo.RedisConfig) (*godo.Response, error) {
	panic("not implemented")
}

// UpdateMySQLConfig ...
func (f *FakeDatabasesService) UpdateMySQLConfig(_ context.Context, _ string, _ *godo.MySQLConfig) (*godo.Response, error) {
	panic("not implemented")
}

// ListOptions ...
func (f *FakeDatabasesService) ListOptions(todo context.Context) (*godo.DatabaseOptions, *godo.Response, error) {
	return f.Options, okResponse, nil
}

// ListTopic ...
func (f *FakeDatabasesService) ListTopics(context.Context, string, *godo.ListOptions) ([]godo.DatabaseTopic, *godo.Response, error) {
	panic("not implemented")
}

// CreateTopic ...
func (f *FakeDatabasesService) CreateTopic(context.Context, string, *godo.DatabaseCreateTopicRequest) (*godo.DatabaseTopic, *godo.Response, error) {
	panic("not implemented")
}

// GetTopic ...
func (f *FakeDatabasesService) GetTopic(context.Context, string, string) (*godo.DatabaseTopic, *godo.Response, error) {
	panic("not implemented")
}

// DeleteTopic ...
func (f *FakeDatabasesService) DeleteTopic(context.Context, string, string) (*godo.Response, error) {
	panic("not implemented")
}

// UpdateTopic ...
func (f *FakeDatabasesService) UpdateTopic(context.Context, string, string, *godo.DatabaseUpdateTopicRequest) (*godo.Response, error) {
	panic("not implemented")
}

// GetMetricsCredentials ...
func (f *FakeDatabasesService) GetMetricsCredentials(ctx context.Context) (*godo.DatabaseMetricsCredentials, *godo.Response, error) {
	panic("not implemented")
}

// UpdateMetricsCredentials ...
func (f *FakeDatabasesService) UpdateMetricsCredentials(ctx context.Context, updateCreds *godo.DatabaseUpdateMetricsCredentialsRequest) (*godo.Response, error) {
	panic("not implemented")
}

// ListDatabaseEvents ...
func (f *FakeDatabasesService) ListDatabaseEvents(ctx context.Context, databaseID string, opts *godo.ListOptions) ([]godo.DatabaseEvent, *godo.Response, error) {
	panic("not implemented")
}

// ListIndexes...
func (f *FakeDatabasesService) ListIndexes(context.Context, string, *godo.ListOptions) ([]godo.DatabaseIndex, *godo.Response, error) {
	panic("not implemented")
}

// DeleteIndex...
func (f *FakeDatabasesService) DeleteIndex(context.Context, string, string) (*godo.Response, error) {
	panic("not implemented")
}

// CreateLogsink...
func (f *FakeDatabasesService) CreateLogsink(context.Context, string, *godo.DatabaseCreateLogsinkRequest) (*godo.DatabaseLogsink, *godo.Response, error) {
	panic("not implemented")
}

// GetLogsink...
func (f *FakeDatabasesService) GetLogsink(context.Context, string, string) (*godo.DatabaseLogsink, *godo.Response, error) {
	panic("not implemented")
}

// ListLogsinks...
func (f *FakeDatabasesService) ListLogsinks(context.Context, string, *godo.ListOptions) ([]godo.DatabaseLogsink, *godo.Response, error) {
	panic("not implemented")
}

// UpdateLogsink...
func (f *FakeDatabasesService) UpdateLogsink(context.Context, string, string, *godo.DatabaseUpdateLogsinkRequest) (*godo.Response, error) {
	panic("not implemented")
}

// DeleteLogsink...
func (f *FakeDatabasesService) DeleteLogsink(context.Context, string, string) (*godo.Response, error) {
	panic("not implemented")
}
