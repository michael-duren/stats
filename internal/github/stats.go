package github

import (
	"context"
	"math"
	"net/url"
)

// Stats is the aggregated profile data backing the stats card.
type Stats struct {
	Name             string
	Username         string
	TotalStars       int
	TotalCommits     int
	TotalPRs         int
	TotalPRsMerged   int
	MergedPRPercent  float64
	TotalReviews     int
	TotalIssues      int
	TotalDiscStarted int
	TotalDiscAnswers int
	ContributedTo    int
	Followers        int
	Rank             Rank
}

// Rank mirrors github-readme-stats: a level (S, A+, ...) plus the percentile
// the user sits at (lower percentile = better).
type Rank struct {
	Level      string
	Percentile float64
}

const statsQuery = `
query userInfo($login: String!, $after: String) {
  user(login: $login) {
    name
    login
    contributionsCollection {
      totalCommitContributions
      totalPullRequestReviewContributions
      restrictedContributionsCount
    }
    repositoriesContributedTo(first: 1, contributionTypes: [COMMIT, ISSUE, PULL_REQUEST, REPOSITORY]) {
      totalCount
    }
    pullRequests(first: 1) { totalCount }
    mergedPullRequests: pullRequests(states: MERGED) { totalCount }
    openIssues: issues(states: OPEN) { totalCount }
    closedIssues: issues(states: CLOSED) { totalCount }
    followers { totalCount }
    repositoryDiscussions { totalCount }
    repositoryDiscussionComments(onlyAnswers: true) { totalCount }
    repositories(first: 100, ownerAffiliations: OWNER, orderBy: {direction: DESC, field: STARGAZERS}, after: $after) {
      totalCount
      nodes { stargazerCount }
      pageInfo { hasNextPage endCursor }
    }
  }
}`

type statsResponse struct {
	User struct {
		Name                    string `json:"name"`
		Login                   string `json:"login"`
		ContributionsCollection struct {
			TotalCommitContributions            int `json:"totalCommitContributions"`
			TotalPullRequestReviewContributions int `json:"totalPullRequestReviewContributions"`
			RestrictedContributionsCount        int `json:"restrictedContributionsCount"`
		} `json:"contributionsCollection"`
		RepositoriesContributedTo struct {
			TotalCount int `json:"totalCount"`
		} `json:"repositoriesContributedTo"`
		PullRequests       struct{ TotalCount int } `json:"pullRequests"`
		MergedPullRequests struct{ TotalCount int } `json:"mergedPullRequests"`
		OpenIssues         struct{ TotalCount int } `json:"openIssues"`
		ClosedIssues       struct{ TotalCount int } `json:"closedIssues"`
		Followers          struct{ TotalCount int } `json:"followers"`
		RepositoryDiscussions        struct{ TotalCount int } `json:"repositoryDiscussions"`
		RepositoryDiscussionComments struct{ TotalCount int } `json:"repositoryDiscussionComments"`
		Repositories       struct {
			TotalCount int `json:"totalCount"`
			Nodes      []struct {
				StargazerCount int `json:"stargazerCount"`
			} `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"repositories"`
	} `json:"user"`
}

// FetchStats pulls and aggregates a user's profile stats.
//
// includeAllCommits switches commit counting from "this year" (the GraphQL
// contributionsCollection, free) to "all time" (a REST commit-search call,
// pricier and approximate). countPrivate folds restricted/private commit
// contributions into the commit total.
func (c *Client) FetchStats(ctx context.Context, username string, includeAllCommits, countPrivate bool) (*Stats, error) {
	var (
		totalStars int
		after      *string
		base       statsResponse
		first      = true
	)
	// Paginate owned repos to sum every stargazer count.
	for {
		var page statsResponse
		vars := map[string]any{"login": username}
		if after != nil {
			vars["after"] = *after
		}
		if err := c.graphql(ctx, statsQuery, vars, &page); err != nil {
			return nil, err
		}
		if first {
			base = page
			first = false
		}
		for _, n := range page.User.Repositories.Nodes {
			totalStars += n.StargazerCount
		}
		if !page.User.Repositories.PageInfo.HasNextPage {
			break
		}
		cur := page.User.Repositories.PageInfo.EndCursor
		after = &cur
	}

	u := base.User
	commits := u.ContributionsCollection.TotalCommitContributions
	if countPrivate {
		commits += u.ContributionsCollection.RestrictedContributionsCount
	}
	if includeAllCommits {
		if all, err := c.fetchTotalCommits(ctx, username); err == nil {
			commits = all
		}
	}

	prs := u.PullRequests.TotalCount
	mergedPRs := u.MergedPullRequests.TotalCount
	issues := u.OpenIssues.TotalCount + u.ClosedIssues.TotalCount
	reviews := u.ContributionsCollection.TotalPullRequestReviewContributions

	mergedPct := 0.0
	if prs > 0 {
		mergedPct = float64(mergedPRs) / float64(prs) * 100
	}

	s := &Stats{
		Name:             firstNonEmpty(u.Name, u.Login, username),
		Username:         username,
		TotalStars:       totalStars,
		TotalCommits:     commits,
		TotalPRs:         prs,
		TotalPRsMerged:   mergedPRs,
		MergedPRPercent:  mergedPct,
		TotalReviews:     reviews,
		TotalIssues:      issues,
		TotalDiscStarted: u.RepositoryDiscussions.TotalCount,
		TotalDiscAnswers: u.RepositoryDiscussionComments.TotalCount,
		ContributedTo:    u.RepositoriesContributedTo.TotalCount,
		Followers:        u.Followers.TotalCount,
	}
	s.Rank = calculateRank(commits, prs, issues, reviews, totalStars, s.Followers)
	return s, nil
}

// fetchTotalCommits uses the REST commit search to approximate lifetime commits.
func (c *Client) fetchTotalCommits(ctx context.Context, username string) (int, error) {
	var out struct {
		TotalCount int `json:"total_count"`
	}
	q := url.Values{}
	q.Set("q", "author:"+username)
	q.Set("per_page", "1")
	if err := c.restGET(ctx, "/search/commits?"+q.Encode(), &out); err != nil {
		return 0, err
	}
	return out.TotalCount, nil
}

// calculateRank ports github-readme-stats' percentile model: a weighted blend
// of CDFs over each metric, then bucketed into letter levels.
func calculateRank(commits, prs, issues, reviews, stars, followers int) Rank {
	const (
		commitsMedian, commitsWeight     = 1000.0, 2.0
		prsMedian, prsWeight             = 50.0, 3.0
		issuesMedian, issuesWeight       = 25.0, 1.0
		reviewsMedian, reviewsWeight     = 2.0, 1.0
		starsMedian, starsWeight         = 50.0, 4.0
		followersMedian, followersWeight = 10.0, 1.0
	)
	totalWeight := commitsWeight + prsWeight + issuesWeight + reviewsWeight + starsWeight + followersWeight

	expCDF := func(x float64) float64 { return 1 - math.Pow(2, -x) }
	logNormalCDF := func(x float64) float64 { return x / (1 + x) }

	rank := 1 - (commitsWeight*expCDF(float64(commits)/commitsMedian)+
		prsWeight*expCDF(float64(prs)/prsMedian)+
		issuesWeight*expCDF(float64(issues)/issuesMedian)+
		reviewsWeight*expCDF(float64(reviews)/reviewsMedian)+
		starsWeight*logNormalCDF(float64(stars)/starsMedian)+
		followersWeight*logNormalCDF(float64(followers)/followersMedian))/totalWeight

	thresholds := []float64{1, 12.5, 25, 37.5, 50, 62.5, 75, 87.5, 100}
	levels := []string{"S", "A+", "A", "A-", "B+", "B", "B-", "C+", "C"}
	pct := rank * 100
	level := levels[len(levels)-1]
	for i, t := range thresholds {
		if pct <= t {
			level = levels[i]
			break
		}
	}
	return Rank{Level: level, Percentile: pct}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
