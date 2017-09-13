// Copyright 2017 orijtech. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcla

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/orijtech/otils"
)

type Event string

const (
	EventIssues      Event = "issues"
	EventPush        Event = "push"
	EventPullRequest Event = "pull_request"
)

type Client struct {
	mu sync.RWMutex
	rt http.RoundTripper

	apiKey string
}

type PullRequestEvent struct {
	Action  Action  `json:"action,omitempty"`
	Number  uint64  `json:"number,omitempty"`
	Changes *Change `json:"changes,omitempty"`

	Repository   *Repository   `json:"repository,omitempty"`
	Sender       *User         `json:"sender,omitempty"`
	Installation *Installation `json:"installation,omitempty"`
}

// PullRequestReviewEvent is the payload that's sent when
// webhook event name "pull_request_review" is fired.
type PullRequestReviewEvent struct {
	Action      Action       `json:"action,omitempty"`
	Changes     *Change      `json:"changes,omitempty"`
	Review      *Review      `json:"review,omitempty"`
	PullRequest *PullRequest `json:"pull_request,omitempty"`
	Repository  *Repository  `json:"repository,omitempty"`
	Sender      *User        `json:"sender,omitempty"`
}

// PullRequestReviewCommentEvent is the payload that's sent
// when webhook event name "pull_request_review_comment" is fired.
type PullRequestReviewCommentEvent struct {
	Action      Action       `json:"action,omitempty"`
	Changes     *Change      `json:"changes,omitempty"`
	PullRequest *PullRequest `json:"pull_request,omitempty"`
	Comment     *Comment     `json:"comment,omitempty"`
}

// PushEvent is the API payload sent when webhook event "push" is fired.
type PushEvent struct {
	// Ref is the full Git ref that was pushed. Example: "refs/heads/master".
	Ref string `json:"ref,omitempty"`
	// Head is the SHA of the most recent commit on ref after the push.
	Head string `json:"head,omitempty"`
	// Before is the SHA of the most recent commit on ref before the push.
	Before              string `json:"before,omitempty"`
	CommitCount         uint64 `json:"size,omitempty"`
	DistinctCommitCount uint64 `json:"distinct_size,omitempty"`

	// Commits describes the pushed commits. The array includes a maximum of
	// 20 commits. If necessary, you can use the Commits API
	// at https://developer.github.com/v3/repos/commits/
	// to fetch additional commits. This limit is applied to timeline
	// events only and isn't applied to webhook deliveries.
	Commits []*Commit `json:"commits,omitempty"`

	HeadCommit *Commit     `json:"head_commit,omitempty"`
	Repository *Repository `json:"repository,omitempty"`
	Pusher     *Author     `json:"pusher,omitempty"`
	Sender     *User       `json:"sender,omitempty"`
}

// ReleaseEvent is the payload sent with a release
// is published and webhook "release" is fired.
type ReleaseEvent struct {
	Action     Action      `json:"action,omitempty"`
	Release    *Release    `json:"release,omitempty"`
	Repository *Repository `json:"repository,omitempty"`
	Sender     *User       `json:"sender,omitempty"`
}

type Release struct {
	URL             string               `json:"url,omitempty"`
	AssetsURL       string               `json:"assets_url,omitempty"`
	UploadURL       string               `json:"upload_url,omitempty"`
	HTMLURL         string               `json:"html_url,omitempty"`
	ID              uint64               `json:"id,omitempty"`
	TagName         string               `json:"tag_name,omitempty"`
	TargetCommitish string               `json:"target_commitish,omitempty"`
	Name            otils.NullableString `json:"name,omitempty"`
	Draft           bool                 `json:"draft,omitempty"`
	Author          *User                `json:"author,omitempty"`
	Prerelease      bool                 `json:"prelease,omitempty"`
	CreatedAt       *time.Time           `json:"created_at,omitempty"`
	PublishedAt     *time.Time           `json:"published_at,omitempty"`
	Assets          []string             `json:"assets,omitempty"`
	TarURL          string               `json:"tarball_url,omitempty"`
	ZipURL          string               `json:"zipball_url,omitempty"`
	Body            otils.NullableString `json:"body,omitempty"`
}

// RepositoryEvent is the payload when webhook event "repository" is fired.
// This event payload is sent when a repository is:
// + created
// + made public
// + made private
//
// Organization hooks are also triggered
// when a repository is deleted.
// Events of this type are not visible in timelines. These events are only
// used to trigger hooks.
type RepositoryEvent struct {
	Action       Action        `json:"action,omitempty"`
	Repository   *Repository   `json:"repository,omitempty"`
	Organization *Organization `json:"organization,omitempty"`
	Sender       *User         `json:"sender,omitempty"`
}

type Organization struct {
	Login            string `json:"login,omitempty"`
	ID               uint64 `json:"id,omitempty"`
	URL              string `json:"url,omitempty"`
	ReposURL         string `json:"repos_url,omitempty"`
	EventsURL        string `json:"events_url,omitempty"`
	MembersURL       string `json:"members_url,omitempty"`
	PublicMembersURL string `json:"public_members_url,omitempty"`
	AvatarsURL       string `json:"avatars_url,omitempty"`
}

// StatusEvent is the payload sent when webhook "status" is fired.
// This event is triggered when the status of a Git commit changes.
// Events of this type are not visible in timelines. These events
// are only used to trigger hooks.
type StatusEvent struct {
	SHA         string    `json:"sha,omitempty"`
	State       State     `json:"state,omitempty"`
	Description string    `json:"description,omitempty"`
	TargetURL   string    `json:"target_url,omitempty"`
	Branches    []*Branch `json:"branches,omitempty"`
	Commit      *Commit   `json:"commit,omitempty"`
	Sender      *User     `json:"sender,omitempty"`
}

type Branch struct {
	Name   string  `json:"name,omitempty"`
	Commit *Commit `json:"commit,omitempty"`
}

type Commit struct {
	ID        string     `json:"id,omitempty"`
	TreeID    string     `json:"tree_id,omitempty"`
	Timestamp *time.Time `json:"timestamp,omitempty"`
	SHA       string     `json:"sha,omitempty"`
	Commit    *Commit    `json:"commit,omitempty"`
	Message   string     `json:"message,omitempty"`
	Author    *Author    `json:"author,omitempty"`
	URL       string     `json:"url,omitempty"`
	HTMLURL   string     `json:"html_url,omitempty"`
	Distinct  bool       `json:"distinct,omitempty"`

	Added    []string `json:"added,omitempty"`
	Removed  []string `json:"removed,omitempty"`
	Modified []string `json:"modified,omitempty"`
}

// TeamEvent is the payload sent when webhook "team" is fired.
// This event is triggered when an organization's team is created or deleted.
// Events of this type are not visible in timelines.
// These events are only used to trigger organization hooks.
type TeamEvent struct {
	Action     Action      `json:"action,omitempty"`
	Team       *Team       `json:"team,omitempty"`
	Changes    *Change     `json:"changes,omitempty"`
	Repository *Repository `json:"repository,omitempty"`
}

// TeamAddEvent is the payload sent when webhook "team_add" is fired.
// This event is triggered when a repository is added to a team.
type TeamAddEvent struct {
	Team       *Team       `json:"team,omitempty"`
	Repository *Repository `json:"repository,omitempty"`
}

// WatchEvent is the payload sent related to starring a repository, not watching.
// Read https://developer.github.com/changes/2012-09-05-watcher-api/ for an explanation.
//
// It is sent when webhook "watch" is fired.
// The event's actor is the user who starred a repository and the event's repository
// is the repository that was starred.
type WatchEvent struct {
	Action string `json:"action,omitempty"`

	Repository *Repository `json:"repository,omitempty"`
	Sender     *User       `json:"sender,omitempty"`
}

type Author struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type Review struct {
	ID   uint64 `json:"id,omitempty"`
	User *User  `json:"user,omitempty"`
	Body string `json:"body,omitempty"`

	SubmittedAt    *time.Time `json:"submitted_at,omitempty"`
	State          State      `json:"state,omitempty"`
	HTMLURL        string     `json:"html_url,omitempty"`
	PullRequestURL string     `json:"pull_request_url,omitempty"`
	Links          *Links     `json:"_links,omitempty"`
}

type State string

const (
	StateActive   State = "active"
	StateApproved State = "approved"
	StateOpen     State = "open"
	StateSuccess  State = "success"
)

type Change struct {
	Title       string  `json:"title,omitempty"`
	Body        string  `json:"body,omitempty"`
	Description string  `json:"description,omitempty"`
	Name        string  `json:"name,omitempty"`
	Privacy     Privacy `json:"privacy,omitempty"`
}

type Privacy string

const (
	PrivacyEdited Privacy = "edited"
	PrivacyPublic Privacy = "public"
	PrivacySecret Privacy = "secret"
)

type PullRequest struct {
	URL            string               `json:"url,omitempty"`
	ID             uint64               `json:"id,omitempty"`
	HTMLURL        string               `json:"html_url,omitempty"`
	DiffURL        string               `json:"diff_url,omitempty"`
	PatchURL       string               `json:"patch_url,omitempty"`
	IssueURL       string               `json:"issue_url,omitempty"`
	Number         uint64               `json:"number,omitempty"`
	State          State                `json:"state,omitempty"`
	Locked         bool                 `json:"locked,omitempty"`
	Title          string               `json:"title,omitempty"`
	User           *User                `json:"user,omitempty"`
	Body           string               `json:"body,omitempty"`
	CreatedAt      *time.Time           `json:"created_at,omitempty"`
	UpdatedAt      *time.Time           `json:"updated_at,omitempty"`
	ClosedAt       *time.Time           `json:"closed_at,omitempty"`
	MergedAt       *time.Time           `json:"merged_at,omitempty"`
	MergeCommitSHA otils.NullableString `json:"merge_commit_sha,omitempty"`
	Assignee       *User                `json:"assignee,omitempty"`
	Milestone      *Milestone           `json:"milestone,omitempty"`
	CommitsURL     string               `json:"commits_url,omitempty"`

	ReviewCommentURL  otils.NullableString `json:"review_comment_url,omitempty"`
	ReviewCommentsURL otils.NullableString `json:"review_comments_url,omitempty"`
	CommentsURL       otils.NullableString `json:"comments_url,omitempty"`
	StatusesURL       otils.NullableString `json:"statuses_url,omitempty"`

	Head *Head `json:"base,omitempty"`
	Base *Head `json:"base,omitempty"`

	Links          *Links               `json:"_links,omitempty"`
	Merged         bool                 `json:"merged,omitempty"`
	Mergeable      otils.NullableString `json:"mergeable,omitempty"`
	MergedBy       *User                `json:"merged_by,omitempty"`
	Comments       uint64               `json:"comments,omitempty"`
	ReviewComments uint64               `json:"review_comments,omitempty"`
	Commits        uint64               `json:"commits,omitempty"`
	Additions      uint64               `json:"additions,omitempty"`
	Deletions      uint64               `json:"deletions,omitempty"`
	ChangedFiles   uint64               `json:"changed_files,omitempty"`
}

type Head struct {
	Label string      `json:"label,omitempty"`
	Ref   string      `json:"ref,omitempty"`
	SHA   string      `json:"sha,omitempty"`
	User  *User       `json:"user,omitempty"`
	Repo  *Repository `json:"repo,omitempty"`
}

type Type string

const (
	TypeUser         Type = "User"
	TypeOrganization Type = "Organization"
	TypeApp          Type = "App"
)

type User struct {
	Username          string `json:"login,omitempty"`
	ID                int64  `json:"id,omitempty"`
	AvatarURL         string `json:"avatar_url,omitempty"`
	GravatarID        string `json:"gravatar_id,omitempty"`
	URL               string `json:"url,omitempty"`
	HTMLURL           string `json:"html_url,omitempty"`
	FollowersURL      string `json:"followers_url,omitempty"`
	GistsURL          string `json:"gists_url,omitempty"`
	StarredURL        string `json:"starred_url,omitempty"`
	SubscriptionsURL  string `json:"subscriptions_url,omitempty"`
	OrganizationURL   string `json:"organization_url,omitempty"`
	ReposURL          string `json:"repos_url,omitempty"`
	EventsURL         string `json:"events_url,omitempty"`
	ReceivedEventsURL string `json:"received_events_url,omitempty"`
	Type              Type   `json:"type,omitempty"`
	SiteAdmin         bool   `json:"site_admin,omitempty"`
}

type Repository struct {
	ID               int64                `json:"id,omitempty"`
	Name             string               `json:"name,omitempty"`
	FullName         string               `json:"full_name,omitempty"`
	Owner            *User                `json:"owner,omitempty"`
	Private          bool                 `json:"private,omitempty"`
	HTMLURL          string               `json:"html_url,omitempty"`
	Description      string               `json:"description,omitempty"`
	Fork             bool                 `json:"fork,omitempty"`
	URL              string               `json:"url,omitempty"`
	ForksURL         string               `json:"forks_url,omitempty"`
	KeysURL          string               `json:"keys_url,omitempty"`
	CollaboratorsURL string               `json:"collaborators_url,omitempty"`
	TeamsURL         string               `json:"teams_url,omitempty"`
	HooksURL         string               `json:"hooks_url,omitempty"`
	IssueEventsURL   string               `json:"issue_events_url,omitempty"`
	EventsURL        string               `json:"events_url,omitempty"`
	AssigneesURL     string               `json:"assignees_url,omitempty"`
	BranchesURL      string               `json:"branches_url,omitempty"`
	TagsURL          string               `json:"tags_url,omitempty"`
	BlobsURL         string               `json:"blobs_url,omitempty"`
	GitTagsURL       string               `json:"git_tags_url,omitempty"`
	GitRefsURL       string               `json:"git_refs_url,omitempty"`
	TreesURL         string               `json:"trees_url,omitempty"`
	StatusURL        string               `json:"statuses_url,omitempty"`
	LanguagesURL     string               `json:"languages_url,omitempty"`
	StargazersURL    string               `json:"stargazers_url,omitempty"`
	ContributorsURL  string               `json:"contributors_url,omitempty"`
	SubscribersURL   string               `json:"subscribers_url,omitempty"`
	SubscriptionURL  string               `json:"subscription_url,omitempty"`
	CommitsURL       string               `json:"commits_url,omitempty"`
	GitCommitsURL    string               `json:"git_commits_url,omitempty"`
	CommentsURL      string               `json:"comments_url,omitempty"`
	IssueCommentURL  string               `json:"issue_comment_url,omitempty"`
	ContentsURL      string               `json:"contents_url,omitempty"`
	CompareURL       string               `json:"compare_url,omitempty"`
	MergesURL        string               `json:"merges_url,omitempty"`
	ArchiveURL       string               `json:"archive_url,omitempty"`
	DownloadsURL     string               `json:"downloads_url,omitempty"`
	IssuesURL        string               `json:"issues_url,omitempty"`
	PullsURL         string               `json:"pulls_url,omitempty"`
	MilestonesURL    string               `json:"milestones_url,omitempty"`
	NotificationsURL string               `json:"notifications_url,omitempty"`
	LabelsURL        string               `json:"labels_url,omitempty"`
	ReleasesURL      string               `json:"releases_url,omitempty"`
	CreatedAt        *time.Time           `json:"created_at,omitempty"`
	UpdatedAt        *time.Time           `json:"updated_at,omitempty"`
	PushedAt         *time.Time           `json:"pushed_at,omitempty"`
	GitURL           string               `json:"git_url,omitempty"`
	SSHURL           string               `json:"ssh_url,omitempty"`
	CloneURL         string               `json:"clone_url,omitempty"`
	SVNURL           string               `json:"svn_url,omitempty"`
	Homepage         otils.NullableString `json:"homepage,omitempty"`
	Size             uint64               `json:"size,omitempty"`
	StargazersCount  uint64               `json:"stargazers_count,omitempty"`
	WatchersCount    uint64               `json:"watchers_count,omitempty"`
	Language         otils.NullableString `json:"language,omitempty"`
	HasIssues        bool                 `json:"has_issues,omitempty"`
	HasDownloads     bool                 `json:"has_downloads,omitempty"`
	HasWiki          bool                 `json:"has_wiki,omitempty"`
	HasPages         bool                 `json:"has_pages,omitempty"`
	ForksCount       uint64               `json:"forks_count,omitempty"`
	MirrorURL        otils.NullableString `json:"mirror_url,omitempty"`
	OpenIssuesCount  uint64               `json:"open_issues_count,omitempty"`
	Forks            uint64               `json:"forks,omitempty"`
	OpenIssueCount   uint64               `json:"open_issues,omitempty"`
	Watchers         uint64               `json:"watchers,omitempty"`
	DefaultBranch    string               `json:"default_branch,omitempty"`
}

type Links struct {
	Self           *Links `json:"self,omitempty"`
	HTML           *Links `json:"html,omitempty"`
	Issue          *Links `json:"issue,omitempty"`
	Comments       *Links `json:"comments,omitempty"`
	ReviewComments *Links `json:"review_comments,omitempty"`
	ReviewComment  *Links `json:"review_comment,omitempty"`
	Commits        *Links `json:"commits,omitempty"`
	Statuses       *Links `json:"statuses,omitempty"`
}

type Installation struct {
	ID uint64 `json:"id,omitempty"`
}

type Comment struct {
	URL              string     `json:"url,omitempty"`
	ID               uint64     `json:"id,omitempty"`
	DiffHunk         string     `json:"diff_hunk,omitempty"`
	Path             string     `json:"path,omitempty"`
	Position         uint64     `json:"position,omitempty"`
	CommitID         string     `json:"commit_id,omitempty"`
	OriginalCommitID string     `json:"original_commit_id,omitempty"`
	User             *User      `json:"user,omitempty"`
	Body             string     `json:"body,omitempty"`
	CreatedAt        *time.Time `json:"created_at,omitempty"`
	UpdatedAt        *time.Time `json:"updated_at,omitempty"`
	HTMLURL          string     `json:"html_url,omitempty"`
	PullRequestURL   string     `json:"pull_request_url,omitempty"`
	Links            *Links     `json:"_links,omitempty"`
}

type Action string

const (
	ActionAdded         Action = "added"
	ActionBlocked       Action = "blocked"
	ActionChanged       Action = "changed"
	ActionCreated       Action = "created"
	ActionDeleted       Action = "deleted"
	ActionMemberInvited Action = "member_invited"
	ActionOpened        Action = "opened"
	ActionPublished     Action = "published"
	ActionRemoved       Action = "removed"
	ActionStarted       Action = "started"
	ActionSubmitted     Action = "submitted"
)

type Milestone struct {
	URL          string               `json:"url,omitempty"`
	LabelsURL    string               `json:"labels_url,omitempty"`
	ID           uint64               `json:"id,omitempty"`
	Number       uint64               `json:"number,omitempty"`
	Title        string               `json:"title,omitempty"`
	Description  otils.NullableString `json:"description,omitempty"`
	Creator      *User                `json:"creator,omitempty"`
	OpenIssues   uint64               `json:"open_issues,omitempty"`
	ClosedIssues uint64               `json:"closed_issues,omitempty"`
	State        State                `json:"state,omitempty"`
	CreatedAt    *time.Time           `json:"created_at,omitempty"`
	UpdatedAt    *time.Time           `json:"updated_at,omitempty"`
	DueOn        *time.Time           `json:"due_on,omitempty"`
	ClosedAt     *time.Time           `json:"closed_at,omitempty"`
	Repository   *Repository          `json:"repository,omitempty"`
	Organization *Organization        `json:"organization,omitempty"`
	Sender       *User                `json:"sender,omitempty"`
}

type OrganizationEvent struct {
	Action     Action      `json:"action,omitempty"`
	Invitation *Invitation `json:"invitation,omitempty"`
}

type Invitation struct {
	ID         uint64               `json:"id,omitempty"`
	Login      string               `json:"login,omitempty"`
	Email      otils.NullableString `json:"email.omitempty"`
	Role       string               `json:"role,omitempty"`
	Membership *Membership          `json:"membership,omitempty"`
}

type Membership struct {
	URL             string        `json:"url,omitempty"`
	State           State         `json:"state,omitempty"`
	Role            string        `json:"role,omitempty"`
	OrganizationURL string        `json:"organization_url,omitempty"`
	User            *User         `json:"user,omitempty"`
	Organization    *Organization `json:"organization,omitempty"`
	Sender          *User         `json:"sender,omitempty"`
}

type Team struct {
	Name        string  `json:"name,omitempty"`
	ID          uint64  `json:"id,omitempty"`
	Slug        string  `json:"slug,omitempty"`
	Description string  `json:"description,omitempty"`
	Privacy     Privacy `json:"privacy,omitempty"`
}

type Hook struct {
	ID        uint64         `json:"id,omitempty"`
	URL       string         `json:"url,omitempty"`
	TestURL   string         `json:"test_url,omitempty"`
	PingURL   string         `json:"ping_url,omitempty"`
	Name      string         `json:"name,omitempty"`
	Events    []string       `json:"events,omitempty"`
	Active    bool           `json:"active,omitempty"`
	Config    *PayloadConfig `json:"config,omitempty"`
	UpdatedAt *time.Time     `json:"updated_at,omitempty"`
	CreatedAt *time.Time     `json:"created_at,omitempty"`
	Type      Type           `json:"type,omitempty"`
	AppID     string         `json:"app_id,omitempty"`
}

type RepoSubscribeRequest struct {
	Owner string
	Repo  string

	HookSubscription *SubscribeRequest
}

type SubscribeRequest struct {
	Name   string  `json:"name,omitempty"`
	Active bool    `json:"active,omitempty"`
	Events []Event `json:"events,omitempty"`

	Config *PayloadConfig `json:"config,omitempty"`
}

type Subscription struct {
	ID      uint64         `json:"id,omitempty"`
	URL     string         `json:"url,omitempty"`
	TestURL string         `json:"test_url,omitempty"`
	PingURL string         `json:"ping_url,omitempty"`
	Name    string         `json:"name,omitempty"`
	Events  []Event        `json:"events,omitempty"`
	Active  bool           `json:"active,omitempty"`
	Config  *PayloadConfig `json:"config,omitempty"`

	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

type ContentType string

const (
	JSON  ContentType = "json"
	JSONP ContentType = "jsonp"
	XML   ContentType = "xml"
)

type PayloadConfig struct {
	URL string `json:"url,omitempty"`

	ContentType ContentType `json:"content_type,omitempty"`
}

const baseURL = "https://api.github.com"

var (
	blankSubscription = new(Subscription)

	errBlankSubscription = errors.New("no subscription could be parsed")
)

func (c *Client) SubscribeToRepo(rsr *RepoSubscribeRequest) (*Subscription, error) {
	blob, err := json.Marshal(rsr.HookSubscription)
	if err != nil {
		return nil, err
	}
	fullURL := fmt.Sprintf("%s/repos/%s/%s/hooks", baseURL, rsr.Owner, rsr.Repo)
	req, err := http.NewRequest("POST", fullURL, bytes.NewReader(blob))
	if err != nil {
		return nil, err
	}
	blob, _, err = c.doHTTPReq(req)
	if err != nil {
		return nil, err
	}
	subs := new(Subscription)
	if err := json.Unmarshal(blob, subs); err != nil {
		return nil, err
	}
	if reflect.DeepEqual(subs, blankSubscription) {
		return nil, errBlankSubscription
	}
	return subs, nil
}

func (c *Client) doHTTPReq(req *http.Request) ([]byte, http.Header, error) {
	// Ensure that we set the header version in the request
	// as recommended at https://developer.github.com/v3/#current-version
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	c.mu.RLock()
	if apiKey := c.apiKey; apiKey != "" {
		req.Header.Add("Authorization", fmt.Sprintf("token %s", apiKey))
	}
	c.mu.RUnlock()

	res, err := c.httpClient().Do(req)
	if err != nil {
		return nil, nil, err
	}
	if res.Body != nil {
		defer res.Body.Close()
	}
	if !otils.StatusOK(res.StatusCode) {
		return nil, res.Header, errors.New(res.Status)
	}
	blob, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, res.Header, err
	}
	return blob, res.Header, nil
}

func (c *Client) SetHTTPRoundTripper(rt http.RoundTripper) {
	c.mu.Lock()
	c.rt = rt
	c.mu.Unlock()
}

func (c *Client) httpClient() *http.Client {
	c.mu.RLock()
	var rt http.RoundTripper = c.rt
	c.mu.RUnlock()

	return &http.Client{Transport: rt}
}

const gclaEnvKey = "GCLA_GITHUB_API_KEY"

func NewClientFromEnv() (*Client, error) {
	apiKey := otils.EnvOrAlternates(gclaEnvKey)
	if apiKey == "" {
		return nil, fmt.Errorf("expecting %q to have been set in your environment", gclaEnvKey)
	}
	return &Client{apiKey: apiKey}, nil
}
