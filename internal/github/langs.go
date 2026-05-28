package github

import (
	"context"
	"sort"
)

// Language is one aggregated language slice for the top-langs card.
type Language struct {
	Name  string
	Color string
	Size  int // bytes
}

const langsQuery = `
query userInfo($login: String!) {
  user(login: $login) {
    repositories(ownerAffiliations: OWNER, isFork: false, first: 100) {
      nodes {
        name
        languages(first: 10, orderBy: {field: SIZE, direction: DESC}) {
          edges {
            size
            node { color name }
          }
        }
      }
    }
  }
}`

type langsResponse struct {
	User struct {
		Repositories struct {
			Nodes []struct {
				Name      string `json:"name"`
				Languages struct {
					Edges []struct {
						Size int `json:"size"`
						Node struct {
							Color string `json:"color"`
							Name  string `json:"name"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"languages"`
			} `json:"nodes"`
		} `json:"repositories"`
	} `json:"user"`
}

// FetchTopLanguages aggregates language byte-sizes across a user's owned,
// non-fork repos and returns them sorted descending.
func (c *Client) FetchTopLanguages(ctx context.Context, username string) ([]Language, error) {
	var resp langsResponse
	if err := c.graphql(ctx, langsQuery, map[string]any{"login": username}, &resp); err != nil {
		return nil, err
	}

	type agg struct {
		color string
		size  int
	}
	totals := map[string]*agg{}
	for _, repo := range resp.User.Repositories.Nodes {
		for _, e := range repo.Languages.Edges {
			a := totals[e.Node.Name]
			if a == nil {
				a = &agg{color: e.Node.Color}
				totals[e.Node.Name] = a
			}
			a.size += e.Size
		}
	}

	out := make([]Language, 0, len(totals))
	for name, a := range totals {
		color := a.color
		if color == "" {
			color = "#858585"
		}
		out = append(out, Language{Name: name, Color: color, Size: a.size})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Size > out[j].Size })
	return out, nil
}
