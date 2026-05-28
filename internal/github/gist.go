package github

import "context"

// Gist backs the gist card.
type Gist struct {
	Name          string
	Description   string
	Language      string
	LanguageColor string
	Stars         int
	Forks         int
}

// GraphQL exposes gists only via node(id), and the public gist hash is not the
// same as the node id. The REST endpoint takes the hash directly, so we use it
// for the file/owner data and the GraphQL node lookup is skipped.
type gistRESTResponse struct {
	Description string `json:"description"`
	Forks       []any  `json:"forks"`
	Files       map[string]struct {
		Language string `json:"language"`
		Filename string `json:"filename"`
	} `json:"files"`
	Owner struct {
		Login string `json:"login"`
	} `json:"owner"`
}

// FetchGist loads a gist by its hash id. REST does not expose a star count, so
// Stars is best-effort (0) here; forks reflect the returned forks array length.
func (c *Client) FetchGist(ctx context.Context, id string) (*Gist, error) {
	var resp gistRESTResponse
	if err := c.restGET(ctx, "/gists/"+id, &resp); err != nil {
		return nil, err
	}
	g := &Gist{
		Name:        firstNonEmpty(resp.Description, id),
		Description: resp.Description,
		Forks:       len(resp.Forks),
	}
	// Use the first file's language as the gist's primary language.
	for _, f := range resp.Files {
		if f.Language != "" {
			g.Language = f.Language
			break
		}
	}
	if g.LanguageColor == "" {
		g.LanguageColor = "#858585"
	}
	return g, nil
}
