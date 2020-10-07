package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/google/go-github/github"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

const (
	CONFIG_FILE = "config.yaml"
	TAG         = "hacktoberfest"
	USER        = "jdecool"

	STATUT_TOPIC_SHOULD_BE_ADDED   = 1
	STATUT_TOPIC_SHOULD_BE_REMOVED = 2
	STATUT_UNKNOW_REPOSITORY       = 3
)

type Configuration struct {
	AccessToken  string
	Repositories map[string]bool
}

func (c Configuration) getRepositoryStatus(r string, topics []string) int {
	_, e := c.Repositories[r]
	if !e {
		return STATUT_UNKNOW_REPOSITORY
	}

	hasTag := contains(topics, TAG)
	shouldHaveTag := c.Repositories[r]

	if !hasTag && shouldHaveTag {
		return STATUT_TOPIC_SHOULD_BE_ADDED
	}

	if hasTag && !shouldHaveTag {
		return STATUT_TOPIC_SHOULD_BE_REMOVED
	}

	return 0
}

var rootCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Github repostiories",
	Run: func(cmd *cobra.Command, args []string) {
		if !fileExists(CONFIG_FILE) {
			panic(errors.New("Configuration file not exists. Run init command before."))
		}

		c, err := loadConfiguration()
		if err != nil {
			panic(err)
		}

		client := createGithubClient(c)
		repos, _, err := client.Repositories.List(context.Background(), USER, nil)
		if err != nil {
			panic(err)
		}

		for _, repo := range repos {
			name := *repo.FullName

			switch c.getRepositoryStatus(name, repo.Topics) {
			case STATUT_TOPIC_SHOULD_BE_ADDED:
				var newTopics []string = append(repo.Topics, TAG)
				client.Repositories.ReplaceAllTopics(context.Background(), *repo.Owner.Login, *repo.Name, newTopics)

			case STATUT_TOPIC_SHOULD_BE_REMOVED:
				var newTopics []string
				for _, t := range repo.Topics {
					if t != TAG {
						newTopics = append(newTopics, t)
					}
				}
				client.Repositories.ReplaceAllTopics(context.Background(), *repo.Owner.Login, *repo.Name, newTopics)

			case STATUT_UNKNOW_REPOSITORY:
				c.Repositories[*repo.FullName] = contains(repo.Topics, TAG)
			}

			// TODO: update repo
		}

		err = saveConfiguration(c)
		if err != nil {
			panic(err)
		}
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Init project configuration",
	Run: func(cmd *cobra.Command, args []string) {
		if fileExists(CONFIG_FILE) {
			panic(errors.New("Configuration file exists. Please remove it before using this command."))
		}

		c, err := loadConfiguration()
		if err != nil {
			c = Configuration{
				AccessToken:  "",
				Repositories: map[string]bool{},
			}
		}

		client := createGithubClient(c)
		repos, _, err := client.Repositories.List(context.Background(), USER, nil)
		if err != nil {
			panic(err)
		}

		for _, repo := range repos {
			c.Repositories[*repo.FullName] = contains(repo.Topics, TAG)
		}

		err = saveConfiguration(c)
		if err != nil {
			panic(err)
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func createGithubClient(c Configuration) *github.Client {
	var http *http.Client = nil

	if c.AccessToken != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: c.AccessToken},
		)
		http = oauth2.NewClient(context.Background(), ts)
	}

	return github.NewClient(http)
}

func contains(a []string, x string) bool {
	for _, v := range a {
		if v == x {
			return true
		}
	}

	return false
}

func fileExists(file string) bool {
	info, err := os.Stat(file)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}

func loadConfiguration() (Configuration, error) {
	var c Configuration

	d, err := ioutil.ReadFile(CONFIG_FILE)
	if err != nil {
		return c, err
	}

	err = yaml.Unmarshal(d, &c)
	if err != nil {
		return c, err
	}

	return c, nil
}

func saveConfiguration(c Configuration) error {
	d, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(CONFIG_FILE, d, 0644)
	if err != nil {
		return err
	}

	return nil
}
