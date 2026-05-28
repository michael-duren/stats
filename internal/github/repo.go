package github

import "context"

// Repo backs the pinned-repo card.
type Repo struct {
	Name          string
	NameWithOwner string
	Description   string
	Stars         int
	Forks         int
	Language      string
	LanguageColor string
	IsArchived    bool
	IsTemplate    bool
}

const repoQuery = `
query repoInfo($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    name
    nameWithOwner
    description
    isArchived
    isTemplate
    stargazerCount
    forkCount
    primaryLanguage { color name }
  }
}`

type repoResponse struct {
	Repository struct {
		Name            string `json:"name"`
		NameWithOwner   string `json:"nameWithOwner"`
		Description     string `json:"description"`
		IsArchived      bool   `json:"isArchived"`
		IsTemplate      bool   `json:"isTemplate"`
		StargazerCount  int    `json:"stargazerCount"`
		ForkCount       int    `json:"forkCount"`
		PrimaryLanguage *struct {
			Color string `json:"color"`
			Name  string `json:"name"`
		} `json:"primaryLanguage"`
	} `json:"repository"`
}

// FetchRepo loads a single repository for the pin card.
func (c *Client) FetchRepo(ctx context.Context, owner, name string) (*Repo, error) {
	var resp repoResponse
	if err := c.graphql(ctx, repoQuery, map[string]any{"owner": owner, "name": name}, &resp); err != nil {
		return nil, err
	}
	r := resp.Repository
	out := &Repo{
		Name:          r.Name,
		NameWithOwner: r.NameWithOwner,
		Description:   r.Description,
		Stars:         r.StargazerCount,
		Forks:         r.ForkCount,
		IsArchived:    r.IsArchived,
		IsTemplate:    r.IsTemplate,
	}
	if r.PrimaryLanguage != nil {
		out.Language = r.PrimaryLanguage.Name
		out.LanguageColor = r.PrimaryLanguage.Color
	}
	if out.LanguageColor == "" {
		out.LanguageColor = "#858585"
	}
	return out, nil
}
