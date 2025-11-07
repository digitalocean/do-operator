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

	// satisfy interface for unimplemented methods
	godo.DatabasesService
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
			URI:      "postgresql://user:password@host:12345/database?sslmode=require",
			Database: "database",
			Host:     "host",
			Port:     12345,
			User:     "user",
			Password: "password",
			SSL:      true,
		},
		PrivateConnection: &godo.DatabaseConnection{
			URI:      "postgresql://private-user:private-password@private-host:12345/private-database?sslmode=require",
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

// ListOptions ...
func (f *FakeDatabasesService) ListOptions(todo context.Context) (*godo.DatabaseOptions, *godo.Response, error) {
	return f.Options, okResponse, nil
}
