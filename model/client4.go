// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package model

import (
	"fmt"
	"net/http"
	"strings"
)

type Response struct {
	StatusCode    int
	Error         *AppError
	RequestId     string
	Etag          string
	ServerVersion string
}

type Client4 struct {
	Url        string       // The location of the server, for example  "http://localhost:8065"
	ApiUrl     string       // The api location of the server, for example "http://localhost:8065/api/v4"
	HttpClient *http.Client // The http client
	AuthToken  string
	AuthType   string
}

func NewAPIv4Client(url string) *Client4 {
	return &Client4{url, url + API_URL_SUFFIX, &http.Client{}, "", ""}
}

func BuildResponse(r *http.Response) *Response {
	return &Response{
		StatusCode:    r.StatusCode,
		RequestId:     r.Header.Get(HEADER_REQUEST_ID),
		Etag:          r.Header.Get(HEADER_ETAG_SERVER),
		ServerVersion: r.Header.Get(HEADER_VERSION_ID),
	}
}

func (c *Client4) SetOAuthToken(token string) {
	c.AuthToken = token
	c.AuthType = HEADER_TOKEN
}

func (c *Client4) ClearOAuthToken() {
	c.AuthToken = ""
	c.AuthType = HEADER_BEARER
}

func (c *Client4) GetUsersRoute() string {
	return fmt.Sprintf("/users")
}

func (c *Client4) GetUserRoute(userId string) string {
	return fmt.Sprintf(c.GetUsersRoute()+"/%v", userId)
}

func (c *Client4) GetUserByUsernameRoute(userName string) string {
	return fmt.Sprintf(c.GetUsersRoute()+"/username/%v", userName)
}

func (c *Client4) GetUserByEmailRoute(email string) string {
	return fmt.Sprintf(c.GetUsersRoute()+"/email/%v", email)
}

func (c *Client4) GetTeamsRoute() string {
	return fmt.Sprintf("/teams")
}

func (c *Client4) GetTeamRoute(teamId string) string {
	return fmt.Sprintf(c.GetTeamsRoute()+"/%v", teamId)
}

func (c *Client4) GetTeamMemberRoute(teamId, userId string) string {
	return fmt.Sprintf(c.GetTeamRoute(teamId)+"/members/%v", userId)
}

func (c *Client4) GetChannelsRoute() string {
	return fmt.Sprintf("/channels")
}

func (c *Client4) GetChannelRoute(channelId string) string {
	return fmt.Sprintf(c.GetChannelsRoute()+"/%v", channelId)
}

func (c *Client4) GetChannelMembersRoute(channelId string) string {
	return fmt.Sprintf(c.GetChannelRoute(channelId) + "/members")
}

func (c *Client4) GetChannelMemberRoute(channelId, userId string) string {
	return fmt.Sprintf(c.GetChannelMembersRoute(channelId)+"/%v", userId)
}

func (c *Client4) GetPostsRoute() string {
	return fmt.Sprintf("/posts")
}

func (c *Client4) GetPostRoute(postId string) string {
	return fmt.Sprintf(c.GetPostsRoute()+"/%v", postId)
}

func (c *Client4) DoApiGet(url string, etag string) (*http.Response, *AppError) {
	return c.DoApiRequest(http.MethodGet, url, "", etag)
}

func (c *Client4) DoApiPost(url string, data string) (*http.Response, *AppError) {
	return c.DoApiRequest(http.MethodPost, url, data, "")
}

func (c *Client4) DoApiPut(url string, data string) (*http.Response, *AppError) {
	return c.DoApiRequest(http.MethodPut, url, data, "")
}

func (c *Client4) DoApiDelete(url string, data string) (*http.Response, *AppError) {
	return c.DoApiRequest(http.MethodDelete, url, "", "")
}

func (c *Client4) DoApiRequest(method, url, data, etag string) (*http.Response, *AppError) {
	rq, _ := http.NewRequest(method, c.ApiUrl+url, strings.NewReader(data))
	rq.Close = true

	if len(etag) > 0 {
		rq.Header.Set(HEADER_ETAG_CLIENT, etag)
	}

	if len(c.AuthToken) > 0 {
		rq.Header.Set(HEADER_AUTH, c.AuthType+" "+c.AuthToken)
	}

	if rp, err := c.HttpClient.Do(rq); err != nil {
		return nil, NewLocAppError(url, "model.client.connecting.app_error", nil, err.Error())
	} else if rp.StatusCode == 304 {
		return rp, nil
	} else if rp.StatusCode >= 300 {
		defer closeBody(rp)
		return rp, AppErrorFromJson(rp.Body)
	} else {
		return rp, nil
	}
}

// CheckStatusOK is a convenience function for checking the standard OK response
// from the web service.
func CheckStatusOK(r *http.Response) bool {
	m := MapFromJson(r.Body)
	defer closeBody(r)

	if m != nil && m[STATUS] == STATUS_OK {
		return true
	}

	return false
}

// Authentication Section

// LoginById authenticates a user by user id and password.
func (c *Client4) LoginById(id string, password string) (*User, *Response) {
	m := make(map[string]string)
	m["id"] = id
	m["password"] = password
	return c.login(m)
}

// Login authenticates a user by login id, which can be username, email or some sort
// of SSO identifier based on server configuration, and a password.
func (c *Client4) Login(loginId string, password string) (*User, *Response) {
	m := make(map[string]string)
	m["login_id"] = loginId
	m["password"] = password
	return c.login(m)
}

// LoginByLdap authenticates a user by LDAP id and password.
func (c *Client4) LoginByLdap(loginId string, password string) (*User, *Response) {
	m := make(map[string]string)
	m["login_id"] = loginId
	m["password"] = password
	m["ldap_only"] = "true"
	return c.login(m)
}

// LoginWithDevice authenticates a user by login id (username, email or some sort
// of SSO identifier based on configuration), password and attaches a device id to
// the session.
func (c *Client4) LoginWithDevice(loginId string, password string, deviceId string) (*User, *Response) {
	m := make(map[string]string)
	m["login_id"] = loginId
	m["password"] = password
	m["device_id"] = deviceId
	return c.login(m)
}

func (c *Client4) login(m map[string]string) (*User, *Response) {
	if r, err := c.DoApiPost("/users/login", MapToJson(m)); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		c.AuthToken = r.Header.Get(HEADER_TOKEN)
		c.AuthType = HEADER_BEARER
		defer closeBody(r)
		return UserFromJson(r.Body), BuildResponse(r)
	}
}

// Logout terminates the current user's session.
func (c *Client4) Logout() (bool, *Response) {
	if r, err := c.DoApiPost("/users/logout", ""); err != nil {
		return false, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		c.AuthToken = ""
		c.AuthType = HEADER_BEARER

		defer closeBody(r)
		return CheckStatusOK(r), BuildResponse(r)
	}
}

// User Section

// CreateUser creates a user in the system based on the provided user struct.
func (c *Client4) CreateUser(user *User) (*User, *Response) {
	if r, err := c.DoApiPost(c.GetUsersRoute(), user.ToJson()); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return UserFromJson(r.Body), BuildResponse(r)
	}
}

// GetUser returns a user based on the provided user id string.
func (c *Client4) GetUser(userId, etag string) (*User, *Response) {
	if r, err := c.DoApiGet(c.GetUserRoute(userId), etag); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return UserFromJson(r.Body), BuildResponse(r)
	}
}

// GetUserByUsername returns a user based on the provided user name string.
func (c *Client4) GetUserByUsername(userName, etag string) (*User, *Response) {
	if r, err := c.DoApiGet(c.GetUserByUsernameRoute(userName), etag); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return UserFromJson(r.Body), BuildResponse(r)
	}
}

// GetUserByEmail returns a user based on the provided user email string.
func (c *Client4) GetUserByEmail(email, etag string) (*User, *Response) {
	if r, err := c.DoApiGet(c.GetUserByEmailRoute(email), etag); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return UserFromJson(r.Body), BuildResponse(r)
	}
}

// GetUsers returns a page of users on the system. Page counting starts at 0.
func (c *Client4) GetUsers(page int, perPage int, etag string) ([]*User, *Response) {
	query := fmt.Sprintf("?page=%v&per_page=%v", page, perPage)
	if r, err := c.DoApiGet(c.GetUsersRoute()+query, etag); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return UserListFromJson(r.Body), BuildResponse(r)
	}
}

// GetUsersInTeam returns a page of users on a team. Page counting starts at 0.
func (c *Client4) GetUsersInTeam(teamId string, page int, perPage int, etag string) ([]*User, *Response) {
	query := fmt.Sprintf("?in_team=%v&page=%v&per_page=%v", teamId, page, perPage)
	if r, err := c.DoApiGet(c.GetUsersRoute()+query, etag); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return UserListFromJson(r.Body), BuildResponse(r)
	}
}

// GetUsersInChannel returns a page of users on a team. Page counting starts at 0.
func (c *Client4) GetUsersInChannel(channelId string, page int, perPage int, etag string) ([]*User, *Response) {
	query := fmt.Sprintf("?in_channel=%v&page=%v&per_page=%v", channelId, page, perPage)
	if r, err := c.DoApiGet(c.GetUsersRoute()+query, etag); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return UserListFromJson(r.Body), BuildResponse(r)
	}
}

// GetUsersNotInChannel returns a page of users on a team. Page counting starts at 0.
func (c *Client4) GetUsersNotInChannel(teamId, channelId string, page int, perPage int, etag string) ([]*User, *Response) {
	query := fmt.Sprintf("?in_team=%v&not_in_channel=%v&page=%v&per_page=%v", teamId, channelId, page, perPage)
	if r, err := c.DoApiGet(c.GetUsersRoute()+query, etag); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return UserListFromJson(r.Body), BuildResponse(r)
	}
}

// GetUsersByIds returns a list of users based on the provided user ids.
func (c *Client4) GetUsersByIds(userIds []string) ([]*User, *Response) {
	if r, err := c.DoApiPost(c.GetUsersRoute()+"/ids", ArrayToJson(userIds)); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return UserListFromJson(r.Body), BuildResponse(r)
	}
}

// UpdateUser updates a user in the system based on the provided user struct.
func (c *Client4) UpdateUser(user *User) (*User, *Response) {
	if r, err := c.DoApiPut(c.GetUserRoute(user.Id), user.ToJson()); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return UserFromJson(r.Body), BuildResponse(r)
	}
}

// UpdateUserPassword updates a user's password. Must be logged in as the user or be a system administrator.
func (c *Client4) UpdateUserPassword(userId, currentPassword, newPassword string) (bool, *Response) {
	requestBody := map[string]string{"current_password": currentPassword, "new_password": newPassword}
	if r, err := c.DoApiPut(c.GetUserRoute(userId)+"/password", MapToJson(requestBody)); err != nil {
		return false, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return CheckStatusOK(r), BuildResponse(r)
	}
}

// UpdateUserRoles updates a user's roles in the system. A user can have "system_user" and "system_admin" roles.
func (c *Client4) UpdateUserRoles(userId, roles string) (bool, *Response) {
	requestBody := map[string]string{"roles": roles}
	if r, err := c.DoApiPut(c.GetUserRoute(userId)+"/roles", MapToJson(requestBody)); err != nil {
		return false, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return CheckStatusOK(r), BuildResponse(r)
	}
}

// DeleteUser deactivates a user in the system based on the provided user id string.
func (c *Client4) DeleteUser(userId string) (bool, *Response) {
	if r, err := c.DoApiDelete(c.GetUserRoute(userId), ""); err != nil {
		return false, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return CheckStatusOK(r), BuildResponse(r)
	}
}

// SendPasswordResetEmail will send a link for password resetting to a user with the
// provided email.
func (c *Client4) SendPasswordResetEmail(email string) (bool, *Response) {
	requestBody := map[string]string{"email": email}
	if r, err := c.DoApiPost(c.GetUsersRoute()+"/password/reset/send", MapToJson(requestBody)); err != nil {
		return false, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return CheckStatusOK(r), BuildResponse(r)
	}
}

// ResetPassword uses a recovery code to update reset a user's password.
func (c *Client4) ResetPassword(code, newPassword string) (bool, *Response) {
	requestBody := map[string]string{"code": code, "new_password": newPassword}
	if r, err := c.DoApiPost(c.GetUsersRoute()+"/password/reset", MapToJson(requestBody)); err != nil {
		return false, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return CheckStatusOK(r), BuildResponse(r)
	}
}

// Team Section

// CreateTeam creates a team in the system based on the provided team struct.
func (c *Client4) CreateTeam(team *Team) (*Team, *Response) {
	if r, err := c.DoApiPost(c.GetTeamsRoute(), team.ToJson()); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return TeamFromJson(r.Body), BuildResponse(r)
	}
}

// GetTeam returns a team based on the provided team id string.
func (c *Client4) GetTeam(teamId, etag string) (*Team, *Response) {
	if r, err := c.DoApiGet(c.GetTeamRoute(teamId), etag); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return TeamFromJson(r.Body), BuildResponse(r)
	}
}

// GetTeamsForUser returns a list of teams a user is on. Must be logged in as the user
// or be a system administrator.
func (c *Client4) GetTeamsForUser(userId, etag string) ([]*Team, *Response) {
	if r, err := c.DoApiGet(c.GetUserRoute(userId)+"/teams", etag); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return TeamListFromJson(r.Body), BuildResponse(r)
	}
}

// GetTeamMember returns a team member based on the provided team and user id strings.
func (c *Client4) GetTeamMember(teamId, userId, etag string) (*TeamMember, *Response) {
	if r, err := c.DoApiGet(c.GetTeamMemberRoute(teamId, userId), etag); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return TeamMemberFromJson(r.Body), BuildResponse(r)
	}
}

// Channel Section

// CreateChannel creates a channel based on the provided channel struct.
func (c *Client4) CreateChannel(channel *Channel) (*Channel, *Response) {
	if r, err := c.DoApiPost(c.GetChannelsRoute(), channel.ToJson()); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return ChannelFromJson(r.Body), BuildResponse(r)
	}
}

// CreateDirectChannel creates a direct message channel based on the two user
// ids provided.
func (c *Client4) CreateDirectChannel(userId1, userId2 string) (*Channel, *Response) {
	requestBody := []string{userId1, userId2}
	if r, err := c.DoApiPost(c.GetChannelsRoute()+"/direct", ArrayToJson(requestBody)); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return ChannelFromJson(r.Body), BuildResponse(r)
	}
}

// GetChannelMembers gets a page of channel members.
func (c *Client4) GetChannelMembers(channelId string, page, perPage int, etag string) (*ChannelMembers, *Response) {
	query := fmt.Sprintf("?page=%v&per_page=%v", page, perPage)
	if r, err := c.DoApiGet(c.GetChannelMembersRoute(channelId)+query, etag); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return ChannelMembersFromJson(r.Body), BuildResponse(r)
	}
}

// GetChannelMember gets a channel member.
func (c *Client4) GetChannelMember(channelId, userId, etag string) (*ChannelMember, *Response) {
	if r, err := c.DoApiGet(c.GetChannelMemberRoute(channelId, userId), etag); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return ChannelMemberFromJson(r.Body), BuildResponse(r)
	}
}

// GetChannelMembersForUser gets all the channel members for a user on a team.
func (c *Client4) GetChannelMembersForUser(userId, teamId, etag string) (*ChannelMembers, *Response) {
	if r, err := c.DoApiGet(fmt.Sprintf(c.GetUserRoute(userId)+"/teams/%v/channels/members", teamId), etag); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return ChannelMembersFromJson(r.Body), BuildResponse(r)
	}
}

// Post Section

// CreatePost creates a post based on the provided post struct.
func (c *Client4) CreatePost(post *Post) (*Post, *Response) {
	if r, err := c.DoApiPost(c.GetPostsRoute(), post.ToJson()); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return PostFromJson(r.Body), BuildResponse(r)
	}
}

// GetPost gets a single post.
func (c *Client4) GetPost(postId string, etag string) (*Post, *Response) {
	if r, err := c.DoApiGet(c.GetPostRoute(postId), etag); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return PostFromJson(r.Body), BuildResponse(r)
	}
}

// GetPostThread gets a post with all the other posts in the same thread.
func (c *Client4) GetPostThread(postId string, etag string) (*PostList, *Response) {
	if r, err := c.DoApiGet(c.GetPostRoute(postId)+"/thread", etag); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return PostListFromJson(r.Body), BuildResponse(r)
	}
}

// GetPostsForChannel gets a page of posts with an array for ordering for a channel.
func (c *Client4) GetPostsForChannel(channelId string, page, perPage int, etag string) (*PostList, *Response) {
	query := fmt.Sprintf("?page=%v&per_page=%v", page, perPage)
	if r, err := c.DoApiGet(c.GetChannelRoute(channelId)+"/posts"+query, etag); err != nil {
		return nil, &Response{StatusCode: r.StatusCode, Error: err}
	} else {
		defer closeBody(r)
		return PostListFromJson(r.Body), BuildResponse(r)
	}
}

// Files Section
// to be filled in..
