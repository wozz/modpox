package gitlab

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/wozz/modpox/upstream"
)

// Config is used to configure the gitlab upstream
type Config struct {
	Upstream upstream.Upstream
	Host     string
	Token    string
}

type privateGitLabUpstream struct {
	host     string
	client   *http.Client
	upstream upstream.Upstream
}

// NewGitLabUpstream creates a new upstream for a private gitlab instance
func NewGitLabUpstream(config *Config) upstream.Upstream {
	hc := &http.Client{
		Timeout: time.Minute,
		Transport: &gitLabRT{
			token: config.Token,
			host:  config.Host,
			base:  http.DefaultTransport,
		},
	}
	return &privateGitLabUpstream{
		client:   hc,
		host:     config.Host,
		upstream: config.Upstream,
	}
}

func sortedTags(t []tagInfo) semVerList {
	sList := make(semVerList, len(t))
	for i, tag := range t {
		sList[i] = parseTag(tag)
	}
	sort.Sort(sList)
	return sList
}

type modInfo struct {
	Version string
	Time    string
}

func (p *privateGitLabUpstream) list(key string) ([]byte, int, error) {
	tags, err := p.getProjectTags(key)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: privateGitLabUpstream list error", err)
	}
	sList := sortedTags(tags)
	var b bytes.Buffer
	for _, s := range sList {
		b.Write([]byte(s.raw))
		b.Write([]byte("\n"))
	}
	return b.Bytes(), http.StatusOK, nil
}

func (p *privateGitLabUpstream) latest(key string) ([]byte, int, error) {
	tags, err := p.getProjectTags(key)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: privateGitLabUpstream latest error", err)
	}
	if len(tags) == 0 {
		commits, err := p.getProjectCommits(key)
		if err != nil {
			return nil, 0, fmt.Errorf("%w: project commits error", err)
		}
		if len(commits) == 0 {
			return nil, 0, fmt.Errorf("could not find latest")
		}
		var b bytes.Buffer
		err = json.NewEncoder(&b).Encode(modInfo{Version: commits[0].versionTag(), Time: commits[0].Date})
		return b.Bytes(), http.StatusOK, err
	}
	sList := sortedTags(tags)
	if len(sList) == 0 {
		return nil, http.StatusGone, nil
	}
	var b bytes.Buffer
	err = json.NewEncoder(&b).Encode(modInfo{Version: sList[len(sList)-1].raw, Time: sList[len(sList)-1].date})
	return b.Bytes(), http.StatusOK, err
}

func (p *privateGitLabUpstream) zip(key string) ([]byte, int, error) {
	// match in reverse so non-greedy match works correctly
	re := regexp.MustCompile(`^piz\.((?U).*)/v@/`)
	revkey := rev(key)
	revmatches := re.FindStringSubmatch(revkey)
	if len(revmatches) < 2 {
		return nil, 0, fmt.Errorf("could not parse request")
	}
	version := rev(revmatches[1])
	versionRE := regexp.MustCompile(`^v\d+\.\d+\.\d+-\d{14}-([\da-f]{12})$`)
	pseudov := versionRE.FindStringSubmatch(version)
	if len(pseudov) == 2 {
		// set to commit sha
		version = pseudov[1]
	}
	zipFile, err := p.getZip(key, version)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: error getting zip", err)
	}
	return zipFile, http.StatusOK, nil
}

func (p *privateGitLabUpstream) info(key string) ([]byte, int, error) {
	info := struct {
		Version string
		Time    string
	}{}
	re := regexp.MustCompile(`^ofni\.((?U).*)/v@/`)
	revkey := rev(key)
	revmatches := re.FindStringSubmatch(revkey)
	if len(revmatches) < 2 {
		return nil, 0, fmt.Errorf("could not parse request")
	}
	version := rev(revmatches[1])
	info.Version = version
	versionRE := regexp.MustCompile(`^v\d+\.\d+\.\d+-\d{14}-([\da-f]{12})$`)
	pseudov := versionRE.FindStringSubmatch(version)
	if len(pseudov) == 2 {
		version = pseudov[1]
	}
	commit, err := p.getCommit(key, version)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: could not get commit", err)
	}
	info.Time = commit.Date
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(&info); err != nil {
		return nil, 0, fmt.Errorf("%w: json encode error", err)
	}
	return b.Bytes(), http.StatusOK, nil
}

func (p *privateGitLabUpstream) mod(key string) ([]byte, int, error) {
	re := regexp.MustCompile(`^dom\.((?U).*)/v@/`)
	revkey := rev(key)
	revmatches := re.FindStringSubmatch(revkey)
	if len(revmatches) < 2 {
		return nil, 0, fmt.Errorf("could not parse request")
	}
	version := rev(revmatches[1])
	versionRE := regexp.MustCompile(`^v\d+\.\d+\.\d+-\d{14}-([\da-f]{12})$`)
	pseudov := versionRE.FindStringSubmatch(version)
	if len(pseudov) == 2 {
		version = pseudov[1]
	}
	goModFile, err := p.getFile(key, version, "go.mod")
	if err != nil {
		return nil, 0, fmt.Errorf("%w: error getting go.mod", err)
	}
	return goModFile, http.StatusOK, nil
}

func (p *privateGitLabUpstream) Get(key string) ([]byte, int, error) {
	if strings.HasPrefix(key, fmt.Sprintf("/%s", p.host)) {
		log.Printf("query private gitlab for %s", key)
		if strings.HasSuffix(key, "/@v/list") {
			return p.list(key)
		} else if strings.HasSuffix(key, "/@latest") {
			return p.latest(key)
		} else if strings.HasSuffix(key, ".zip") {
			return p.zip(key)
		} else if strings.HasSuffix(key, ".info") {
			return p.info(key)
		} else if strings.HasSuffix(key, ".mod") {
			return p.mod(key)
		}
		return nil, http.StatusForbidden, nil
	}
	return p.upstream.Get(key)
}

type projectInfo struct {
	Id   int    `json:"id"`
	Path string `json:"path_with_namespace"`
}

type commitInfo struct {
	Id   string `json:"id"`
	Date string `json:"created_at"`
}

func (c commitInfo) versionTag() string {
	t, err := time.Parse("2006-01-02T15:04:05.000Z", c.Date)
	if err != nil {
		log.Printf("could not parse date: %s %v", c.Date, err)
		return ""
	}
	id := ""
	if len(c.Id) > 12 {
		id = c.Id[:12]
	} else {
		id = c.Id
	}
	return fmt.Sprintf("v0.0.0-%s-%s", t.Format("20060102150405"), id)
}

func (p *privateGitLabUpstream) getFile(key string, version string, filename string) ([]byte, error) {
	projectPath, err := parseProjectPath(key)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get project path", err)
	}
	projectId, err := p.getProjectId(projectPath)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get project id", err)
	}
	queryParams := &url.Values{
		"ref": []string{version},
	}
	fileApiData, err := p.apiReq(fmt.Sprintf("projects/%d/repository/files/%s?%s", projectId, url.PathEscape(filename), queryParams.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: could not make api req for files", err)
	}
	fileData := struct {
		Content []byte `json:"content"`
	}{}
	err = json.Unmarshal(fileApiData, &fileData)
	return fileData.Content, err
}

func (p *privateGitLabUpstream) getZip(key string, tag string) ([]byte, error) {
	projectPath, err := parseProjectPath(key)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get project path", err)
	}
	projectId, err := p.getProjectId(projectPath)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get project id", err)
	}
	queryParams := &url.Values{
		"sha": []string{tag},
	}
	rawZip, err := p.apiReq(fmt.Sprintf("projects/%d/repository/archive.zip?%s", projectId, queryParams.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: could not make api req", err)
	}
	b := bytes.NewReader(rawZip)
	outBuf := new(bytes.Buffer)
	w := zip.NewWriter(outBuf)
	r, err := zip.NewReader(b, int64(len(rawZip)))
	if err != nil {
		return nil, fmt.Errorf("%w: could not read zip file", err)
	}
	keyPath := strings.TrimPrefix(key, "/")
	keyPath = strings.TrimSuffix(key, ".zip")
	modParts := strings.Split(keyPath, "/@v/")
	if len(modParts) != 2 {
		return nil, fmt.Errorf("could not parse module path for zip file")
	}
	modPath := strings.Join(modParts, "@")
	modPath = strings.TrimPrefix(modPath, "/")
	for _, f := range r.File {
		fileNameParts := strings.Split(f.Name, "/")
		fileNameParts[0] = modPath
		outF, err := w.Create(strings.Join(fileNameParts, "/"))
		if err != nil {
			return nil, fmt.Errorf("%w: could not create new zip file", err)
		}
		inF, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("%w: could not create new file for zip", err)
		}
		io.Copy(outF, inF)
		inF.Close()
	}
	w.Flush()
	w.Close()
	return outBuf.Bytes(), nil
}

func (p *privateGitLabUpstream) getCommit(key, version string) (commitInfo, error) {
	projectPath, err := parseProjectPath(key)
	if err != nil {
		return commitInfo{}, fmt.Errorf("%w: could not parse project path", err)
	}
	projectId, err := p.getProjectId(projectPath)
	if err != nil {
		return commitInfo{}, fmt.Errorf("%w: could not parse project id", err)
	}
	commitData, err := p.apiReq(fmt.Sprintf("projects/%d/repository/commits/%s", projectId, version))
	if err != nil {
		return commitInfo{}, fmt.Errorf("%w: could not make api request for commits", err)
	}
	var c commitInfo
	err = json.Unmarshal(commitData, &c)
	return c, err

}

func (p *privateGitLabUpstream) getProjectCommits(key string) ([]commitInfo, error) {
	projectPath, err := parseProjectPath(key)
	if err != nil {
		return nil, fmt.Errorf("%w: could not parse project path", err)
	}
	projectId, err := p.getProjectId(projectPath)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get project id", err)
	}
	commitData, err := p.apiReq(fmt.Sprintf("projects/%d/repository/commits", projectId))
	if err != nil {
		return nil, fmt.Errorf("%w: could not make api req", err)
	}
	cList := make([]commitInfo, 0)
	err = json.Unmarshal(commitData, &cList)
	return cList, err
}

type tagInfo struct {
	Name   string     `json:"name"`
	Commit commitInfo `json:"commit"`
}

func (p *privateGitLabUpstream) getProjectTags(key string) ([]tagInfo, error) {
	projectPath, err := parseProjectPath(key)
	if err != nil {
		return nil, fmt.Errorf("%w: could not parse project path", err)
	}
	projectId, err := p.getProjectId(projectPath)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get project id", err)
	}
	tagData, err := p.apiReq(fmt.Sprintf("projects/%d/repository/tags", projectId))
	if err != nil {
		return nil, fmt.Errorf("%w: could not make api req", err)
	}
	tList := make([]tagInfo, 0)
	err = json.Unmarshal(tagData, &tList)
	if err != nil {
		return nil, fmt.Errorf("%w: could not parse json response", err)
	}
	return tList, nil
}

func parseProjectPath(key string) (string, error) {
	projectPathParts := strings.Split(key, "/")
	if len(projectPathParts) < 4 {
		return "", fmt.Errorf("invalid path")
	}
	projectPath := projectPathParts[2] + "/" + projectPathParts[3]
	return projectPath, nil
}

func (p *privateGitLabUpstream) getProjectId(projectPath string) (int, error) {
	projects, err := p.apiReq("projects")
	if err != nil {
		return 0, fmt.Errorf("%w: could not get projects", err)
	}
	piList := make([]projectInfo, 0)
	err = json.Unmarshal(projects, &piList)
	if err != nil {
		return 0, fmt.Errorf("%w: could not parse projects json", err)
	}
	for _, pi := range piList {
		if pi.Path == projectPath {
			return pi.Id, nil
		}
	}
	return 0, fmt.Errorf("not found")
}

type gitLabRT struct {
	token string
	host  string
	base  http.RoundTripper
}

func (g *gitLabRT) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Add("Private-Token", g.token)
	if r.URL.Host != g.host {
		return nil, fmt.Errorf("invalid host")
	}
	if r.URL.Scheme != "https" {
		return nil, fmt.Errorf("insecure connection")
	}
	return g.base.RoundTrip(r)
}

func (p *privateGitLabUpstream) apiReq(path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%s/api/v4/%s", p.host, path), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: could not make request", err)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: could not perform request", err)
	}
	var b bytes.Buffer
	defer resp.Body.Close()
	_, err = io.Copy(&b, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: could not copy response", err)
	}
	return b.Bytes(), nil
}

func rev(in string) string {
	out := ""
	for i := range in {
		out = string(in[i]) + out
	}
	return out
}
