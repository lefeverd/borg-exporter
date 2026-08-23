package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestLoad_ValidMixedConfig(t *testing.T) {
	path := writeConfig(t, `
repositories:
  - name: home
    type: borg
    path: /path/to/repo
    opts: "--lock-wait 5"
  - name: photos
    type: restic
    repository: sftp:user@host:/path
    password_command: "pass show restic/photos"
`)

	cfg, err := Load(path)
	require.NoError(t, err)

	require.Len(t, cfg.Repositories, 2)
	assert.Equal(t, Repository{Name: "home", Type: TypeBorg, Path: "/path/to/repo", Opts: "--lock-wait 5"}, cfg.Repositories[0])
	assert.Equal(t, Repository{Name: "photos", Type: TypeRestic, ResticRepository: "sftp:user@host:/path", PasswordCommand: "pass show restic/photos"}, cfg.Repositories[1])

	assert.Equal(t, []Repository{cfg.Repositories[0]}, cfg.BorgRepositories())
	assert.Equal(t, []Repository{cfg.Repositories[1]}, cfg.ResticRepositories())
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	assert.Error(t, err)
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeConfig(t, "not: valid: yaml: [")
	_, err := Load(path)
	assert.Error(t, err)
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		repos   []Repository
		wantErr string
	}{
		{
			name:    "no repositories",
			repos:   nil,
			wantErr: "no repositories configured",
		},
		{
			name:    "missing name",
			repos:   []Repository{{Type: TypeBorg, Path: "/repo"}},
			wantErr: "name",
		},
		{
			name:    "duplicate name",
			repos:   []Repository{{Name: "a", Type: TypeBorg, Path: "/repo1"}, {Name: "a", Type: TypeBorg, Path: "/repo2"}},
			wantErr: "duplicate",
		},
		{
			name:    "unknown type",
			repos:   []Repository{{Name: "a", Type: "unknown"}},
			wantErr: "unknown",
		},
		{
			name:    "borg missing path",
			repos:   []Repository{{Name: "a", Type: TypeBorg}},
			wantErr: "path",
		},
		{
			name:    "restic missing repository",
			repos:   []Repository{{Name: "a", Type: TypeRestic, Password: "x"}},
			wantErr: "repository",
		},
		{
			name:    "restic missing password fields",
			repos:   []Repository{{Name: "a", Type: TypeRestic, ResticRepository: "sftp:x"}},
			wantErr: "password",
		},
		{
			name: "restic multiple password fields",
			repos: []Repository{{
				Name: "a", Type: TypeRestic, ResticRepository: "sftp:x",
				Password: "x", PasswordFile: "/f",
			}},
			wantErr: "password",
		},
		{
			name: "restic password only, valid but discouraged",
			repos: []Repository{{
				Name: "a", Type: TypeRestic, ResticRepository: "sftp:x", Password: "x",
			}},
			wantErr: "",
		},
		{
			name: "restic password_file only, valid",
			repos: []Repository{{
				Name: "a", Type: TypeRestic, ResticRepository: "sftp:x", PasswordFile: "/f",
			}},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Repositories: tt.repos}
			err := cfg.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestRepository_UsesPlaintextPassword(t *testing.T) {
	assert.True(t, Repository{Type: TypeRestic, Password: "x"}.UsesPlaintextPassword())
	assert.False(t, Repository{Type: TypeRestic, PasswordFile: "/f"}.UsesPlaintextPassword())
	assert.False(t, Repository{Type: TypeBorg}.UsesPlaintextPassword())
}
