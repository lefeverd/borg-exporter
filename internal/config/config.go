package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// RepositoryType discriminates which backup tool a Repository entry is for.
type RepositoryType string

const (
	TypeBorg   RepositoryType = "borg"
	TypeRestic RepositoryType = "restic"
)

// Repository is a single configured backup repository, of either type.
// Fields not relevant to the entry's Type are left zero-valued.
type Repository struct {
	Name string         `yaml:"name"`
	Type RepositoryType `yaml:"type"`

	// borg-specific
	Path string `yaml:"path,omitempty"`
	Opts string `yaml:"opts,omitempty"`

	// restic-specific. Exactly one of Password/PasswordFile/PasswordCommand
	// must be set; Password mirrors restic's own RESTIC_PASSWORD convention
	// but is discouraged in favor of the other two.
	ResticRepository string `yaml:"repository,omitempty"`
	Password         string `yaml:"password,omitempty"`
	PasswordFile     string `yaml:"password_file,omitempty"`
	PasswordCommand  string `yaml:"password_command,omitempty"`
}

// UsesPlaintextPassword reports whether this restic repository is configured
// with the discouraged plaintext password field.
func (r Repository) UsesPlaintextPassword() bool {
	return r.Type == TypeRestic && r.Password != ""
}

// Config is the exporter's repository configuration, loaded from YAML.
type Config struct {
	Repositories []Repository `yaml:"repositories"`
}

// Load reads and validates the YAML config file at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// Validate checks that every repository entry is well-formed.
func (c *Config) Validate() error {
	if len(c.Repositories) == 0 {
		return fmt.Errorf("no repositories configured")
	}

	seen := make(map[string]bool, len(c.Repositories))
	for i, repo := range c.Repositories {
		if repo.Name == "" {
			return fmt.Errorf("repositories[%d]: name is required", i)
		}
		if seen[repo.Name] {
			return fmt.Errorf("repositories[%d]: duplicate repository name %q", i, repo.Name)
		}
		seen[repo.Name] = true

		switch repo.Type {
		case TypeBorg:
			if repo.Path == "" {
				return fmt.Errorf("repository %q: path is required for type borg", repo.Name)
			}
		case TypeRestic:
			if repo.ResticRepository == "" {
				return fmt.Errorf("repository %q: repository is required for type restic", repo.Name)
			}
			passwordFieldsSet := 0
			for _, v := range []string{repo.Password, repo.PasswordFile, repo.PasswordCommand} {
				if v != "" {
					passwordFieldsSet++
				}
			}
			if passwordFieldsSet != 1 {
				return fmt.Errorf("repository %q: exactly one of password, password_file, password_command is required", repo.Name)
			}
		default:
			return fmt.Errorf("repository %q: unknown type %q", repo.Name, repo.Type)
		}
	}

	return nil
}

// BorgRepositories returns the configured repositories of type borg.
func (c *Config) BorgRepositories() []Repository {
	return c.filterByType(TypeBorg)
}

// ResticRepositories returns the configured repositories of type restic.
func (c *Config) ResticRepositories() []Repository {
	return c.filterByType(TypeRestic)
}

func (c *Config) filterByType(t RepositoryType) []Repository {
	var repos []Repository
	for _, repo := range c.Repositories {
		if repo.Type == t {
			repos = append(repos, repo)
		}
	}
	return repos
}
