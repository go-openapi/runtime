// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2014 Naoya Inada <naoina@kuune.org>
// SPDX-License-Identifier: MIT

package denco_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/go-openapi/runtime/middleware/denco"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func routes() []denco.Record {
	return []denco.Record{
		{"/", testRoute0},
		{pathPathToRoute, testRoute1},
		{"/path/to/other", testRoute2},
		{"/path/to/route/a", testRoute3},
		{"/path/to/:param", "testroute4"},
		{"/gists/:param1/foo/:param2", "testroute12"},
		{"/gists/:param1/foo/bar", "testroute11"},
		{"/:param1/:param2/foo/:param3", "testroute13"},
		{"/path/to/wildcard/*routepath", "testroute5"},
		{"/path/to/:param1/:param2", "testroute6"},
		{"/path/to/:param1/sep/:param2", "testroute7"},
		{"/:year/:month/:day", "testroute8"},
		{pathUserID, "testroute9"},
		{"/a/to/b/:param/*routepath", "testroute10"},
		{"/path/with/key=:value", "testroute14"},
	}
}

var realURIs = []denco.Record{
	{pathAuthorizations, pathAuthorizations},
	{pathAuthorizationsID, pathAuthorizationsID},
	{pathAppsTokens, pathAppsTokens},
	{pathEvents, pathEvents},
	{pathReposEvents, pathReposEvents},
	{pathNetworksEvents, pathNetworksEvents},
	{pathOrgsEvents, pathOrgsEvents},
	{pathUsersReceivedEvents, pathUsersReceivedEvents},
	{pathUsersReceivedEventsPublic, pathUsersReceivedEventsPublic},
	{pathUsersEvents, pathUsersEvents},
	{pathUsersEventsPublic, pathUsersEventsPublic},
	{pathUsersEventsOrgs, pathUsersEventsOrgs},
	{pathFeeds, pathFeeds},
	{pathNotifications, pathNotifications},
	{pathReposNotifications, pathReposNotifications},
	{pathNotificationThreads, pathNotificationThreads},
	{pathNotificationThreadSub, pathNotificationThreadSub},
	{pathReposStargazers, pathReposStargazers},
	{pathUsersStarred, pathUsersStarred},
	{pathUserStarred, pathUserStarred},
	{pathUserStarredOwnerRepo, pathUserStarredOwnerRepo},
	{pathReposSubscribers, pathReposSubscribers},
	{pathUsersSubscriptions, pathUsersSubscriptions},
	{pathUserSubscriptions, pathUserSubscriptions},
	{pathReposSubscription, pathReposSubscription},
	{pathUserSubscriptionsOwnerRepo, pathUserSubscriptionsOwnerRepo},
	{pathUsersGists, pathUsersGists},
	{pathGists, pathGists},
	{pathGistsID, pathGistsID},
	{pathGistsIDStar, pathGistsIDStar},
	{pathReposGitBlobs, pathReposGitBlobs},
	{pathReposGitCommits, pathReposGitCommits},
	{pathReposGitRefs, pathReposGitRefs},
	{pathReposGitTags, pathReposGitTags},
	{pathReposGitTrees, pathReposGitTrees},
	{pathIssues, pathIssues},
	{pathUserIssues, pathUserIssues},
	{pathOrgsIssues, pathOrgsIssues},
	{pathReposIssues, pathReposIssues},
	{pathReposIssue, pathReposIssue},
	{pathReposAssignees, pathReposAssignees},
	{pathReposAssignee, pathReposAssignee},
	{pathReposIssueComments, pathReposIssueComments},
	{pathReposIssueEvents, pathReposIssueEvents},
	{pathReposLabels, pathReposLabels},
	{pathReposLabel, pathReposLabel},
	{pathReposIssueLabels, pathReposIssueLabels},
	{pathReposMilestoneLabels, pathReposMilestoneLabels},
	{pathReposMilestones, pathReposMilestones},
	{pathReposMilestone, pathReposMilestone},
	{pathEmojis, pathEmojis},
	{pathGitignoreTemplates, pathGitignoreTemplates},
	{pathGitignoreTemplate, pathGitignoreTemplate},
	{pathMeta, pathMeta},
	{pathRateLimit, pathRateLimit},
	{pathUsersOrgs, pathUsersOrgs},
	{pathUserOrgs, pathUserOrgs},
	{pathOrgsOrg, pathOrgsOrg},
	{pathOrgsMembers, pathOrgsMembers},
	{pathOrgsMember, pathOrgsMember},
	{pathOrgsPublicMembers, pathOrgsPublicMembers},
	{pathOrgsPublicMember, pathOrgsPublicMember},
	{pathOrgsTeams, pathOrgsTeams},
	{pathTeamsID, pathTeamsID},
	{pathTeamsMembers, pathTeamsMembers},
	{pathTeamsMember, pathTeamsMember},
	{pathTeamsRepos, pathTeamsRepos},
	{pathTeamsRepo, pathTeamsRepo},
	{pathUserTeams, pathUserTeams},
	{pathReposPulls, pathReposPulls},
	{pathReposPull, pathReposPull},
	{pathReposPullCommits, pathReposPullCommits},
	{pathReposPullFiles, pathReposPullFiles},
	{pathReposPullMerge, pathReposPullMerge},
	{pathReposPullComments, pathReposPullComments},
	{pathUserRepos, pathUserRepos},
	{pathUsersRepos, pathUsersRepos},
	{pathOrgsRepos, pathOrgsRepos},
	{pathRepositories, pathRepositories},
	{pathRepoOwnerRepo, pathRepoOwnerRepo},
	{pathReposContributors, pathReposContributors},
	{pathReposLanguages, pathReposLanguages},
	{pathReposTeams, pathReposTeams},
	{pathReposTags, pathReposTags},
	{pathReposBranches, pathReposBranches},
	{pathReposBranch, pathReposBranch},
	{pathReposCollaborators, pathReposCollaborators},
	{pathReposCollaborator, pathReposCollaborator},
	{pathReposComments, pathReposComments},
	{pathReposCommitsSHAComments, pathReposCommitsSHAComments},
	{pathReposComment, pathReposComment},
	{pathReposCommits, pathReposCommits},
	{pathReposCommit, pathReposCommit},
	{pathReposReadme, pathReposReadme},
	{pathReposKeys, pathReposKeys},
	{pathReposKey, pathReposKey},
	{pathReposDownloads, pathReposDownloads},
	{pathReposDownload, pathReposDownload},
	{pathReposForks, pathReposForks},
	{pathReposHooks, pathReposHooks},
	{pathReposHook, pathReposHook},
	{pathReposReleases, pathReposReleases},
	{pathReposRelease, pathReposRelease},
	{pathReposReleaseAssets, pathReposReleaseAssets},
	{pathReposStatsContributors, pathReposStatsContributors},
	{pathReposStatsCommitActivity, pathReposStatsCommitActivity},
	{pathReposStatsCodeFrequency, pathReposStatsCodeFrequency},
	{pathReposStatsParticipation, pathReposStatsParticipation},
	{pathReposStatsPunchCard, pathReposStatsPunchCard},
	{pathReposStatuses, pathReposStatuses},
	{pathSearchRepositories, pathSearchRepositories},
	{pathSearchCode, pathSearchCode},
	{pathSearchIssues, pathSearchIssues},
	{pathSearchUsers, pathSearchUsers},
	{pathLegacyIssuesSearch, pathLegacyIssuesSearch},
	{pathLegacyReposSearch, pathLegacyReposSearch},
	{pathLegacyUserSearch, pathLegacyUserSearch},
	{pathLegacyUserEmail, pathLegacyUserEmail},
	{pathUsersUser, pathUsersUser},
	{pathUser, pathUser},
	{pathUsers, pathUsers},
	{pathUserEmails, pathUserEmails},
	{pathUsersFollowers, pathUsersFollowers},
	{pathUserFollowers, pathUserFollowers},
	{pathUsersFollowing, pathUsersFollowing},
	{pathUserFollowing, pathUserFollowing},
	{pathUserFollowingUser, pathUserFollowingUser},
	{pathUsersFollowingTarget, pathUsersFollowingTarget},
	{pathUsersKeys, pathUsersKeys},
	{pathUserKeys, pathUserKeys},
	{pathUserKey, pathUserKey},
	{pathPeopleUserID, pathPeopleUserID},
	{pathPeople, pathPeople},
	{pathActivitiesPeople, pathActivitiesPeople},
	{pathPeoplePeople, pathPeoplePeople},
	{pathPeopleOpenIDConnect, pathPeopleOpenIDConnect},
	{pathPeopleActivities, pathPeopleActivities},
	{pathActivitiesActivityID, pathActivitiesActivityID},
	{pathActivities, pathActivities},
	{pathActivitiesComments, pathActivitiesComments},
	{pathCommentsCommentID, pathCommentsCommentID},
	{pathPeopleMoments, pathPeopleMoments},
}

type testcase struct {
	path   string
	value  any
	params []denco.Param
	found  bool
}

func runLookupTest(t *testing.T, records []denco.Record, testcases []testcase) {
	r := denco.New()
	if err := r.Build(records); err != nil {
		t.Fatal(err)
	}
	for _, testcase := range testcases {
		data, params, found := r.Lookup(testcase.path)
		if !reflect.DeepEqual(data, testcase.value) || !reflect.DeepEqual(params, denco.Params(testcase.params)) || !reflect.DeepEqual(found, testcase.found) {
			t.Errorf("Router.Lookup(%q) => (%#v, %#v, %#v), want (%#v, %#v, %#v)", testcase.path, data, params, found, testcase.value, denco.Params(testcase.params), testcase.found)
		}
	}
}

func TestRouter_Lookup(t *testing.T) {
	testcases := []testcase{
		{"/", testRoute0, nil, true},
		{"/gists/1323/foo/bar", "testroute11", []denco.Param{{paramParam1, "1323"}}, true},
		{"/gists/1323/foo/133", "testroute12", []denco.Param{{paramParam1, "1323"}, {paramParam2, "133"}}, true},
		{"/234/1323/foo/133", "testroute13", []denco.Param{{paramParam1, "234"}, {paramParam2, "1323"}, {"param3", "133"}}, true},
		{pathPathToRoute, testRoute1, nil, true},
		{"/path/to/other", testRoute2, nil, true},
		{"/path/to/route/a", testRoute3, nil, true},
		{"/path/to/hoge", "testroute4", []denco.Param{{"param", "hoge"}}, true},
		{"/path/to/wildcard/some/params", "testroute5", []denco.Param{{"routepath", "some/params"}}, true},
		{"/path/to/o1/o2", "testroute6", []denco.Param{{paramParam1, "o1"}, {paramParam2, "o2"}}, true},
		{"/path/to/p1/sep/p2", "testroute7", []denco.Param{{paramParam1, "p1"}, {paramParam2, "p2"}}, true},
		{"/2014/01/06", "testroute8", []denco.Param{{"year", "2014"}, {"month", "01"}, {"day", "06"}}, true},
		{"/user/777", "testroute9", []denco.Param{{paramID, "777"}}, true},
		{"/a/to/b/p1/some/wildcard/params", "testroute10", []denco.Param{{"param", "p1"}, {"routepath", "some/wildcard/params"}}, true},
		{"/missing", nil, nil, false},
		{"/path/with/key=value", "testroute14", []denco.Param{{paramValue, paramValue}}, true},
	}
	runLookupTest(t, routes(), testcases)

	records := []denco.Record{
		{"/", testRoute0},
		{"/:b", testRoute1},
		{"/*wildcard", testRoute2},
	}
	testcases = []testcase{
		{"/", testRoute0, nil, true},
		{"/true", testRoute1, []denco.Param{{"b", "true"}}, true},
		{"/foo/bar", testRoute2, []denco.Param{{"wildcard", "foo/bar"}}, true},
	}
	runLookupTest(t, records, testcases)

	records = []denco.Record{
		{pathNetworksEvents, testRoute0},
		{pathOrgsEvents, testRoute1},
		{pathNotificationThreads, testRoute2},
		{"/mypathisgreat/:thing-id", testRoute3},
	}
	testcases = []testcase{
		{pathNetworksEvents, testRoute0, []denco.Param{{paramOwner, ":owner"}, {paramRepo, ":repo"}}, true},
		{pathOrgsEvents, testRoute1, []denco.Param{{paramOrg, ":org"}}, true},
		{pathNotificationThreads, testRoute2, []denco.Param{{paramID, ":id"}}, true},
		{"/mypathisgreat/:thing-id", testRoute3, []denco.Param{{"thing-id", ":thing-id"}}, true},
	}
	runLookupTest(t, records, testcases)

	runLookupTest(t, []denco.Record{
		{"/", "route2"},
	}, []testcase{
		{pathUserAlice, nil, nil, false},
	})

	runLookupTest(t, []denco.Record{
		{"/user/:name", "route1"},
	}, []testcase{
		{"/", nil, nil, false},
	})

	runLookupTest(t, []denco.Record{
		{"/*wildcard", testRoute0},
		{"/a/:b", testRoute1},
	}, []testcase{
		{"/a", testRoute0, []denco.Param{{"wildcard", "a"}}, true},
	})
}

func TestRouter_Lookup_withManyRoutes(t *testing.T) {
	n := 1000
	records := make([]denco.Record, n)
	for i := range n {
		records[i] = denco.Record{Key: "/" + randomString(rand.Intn(50)+10), Value: fmt.Sprintf("route%d", i)} //#nosec
	}
	router := denco.New()
	require.NoError(t, router.Build(records))

	for _, r := range records {
		data, params, found := router.Lookup(r.Key)
		assert.Equal(t, r.Value, data)
		assert.Empty(t, params)
		assert.TrueT(t, found)
	}
}

func TestRouter_Lookup_realURIs(t *testing.T) {
	testcases := []testcase{
		{pathAuthorizations, pathAuthorizations, nil, true},
		{"/authorizations/1", pathAuthorizationsID, []denco.Param{{paramID, "1"}}, true},
		{"/applications/1/tokens/zohRoo7e", pathAppsTokens, []denco.Param{{"client_id", "1"}, {"access_token", "zohRoo7e"}}, true},
		{pathEvents, pathEvents, nil, true},
		{"/repos/naoina/denco/events", pathReposEvents, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/networks/naoina/denco/events", pathNetworksEvents, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/orgs/something/events", pathOrgsEvents, []denco.Param{{paramOrg, valSomething}}, true},
		{"/users/naoina/received_events", pathUsersReceivedEvents, []denco.Param{{paramUser, valNaoina}}, true},
		{"/users/naoina/received_events/public", pathUsersReceivedEventsPublic, []denco.Param{{paramUser, valNaoina}}, true},
		{"/users/naoina/events", pathUsersEvents, []denco.Param{{paramUser, valNaoina}}, true},
		{"/users/naoina/events/public", pathUsersEventsPublic, []denco.Param{{paramUser, valNaoina}}, true},
		{"/users/naoina/events/orgs/something", pathUsersEventsOrgs, []denco.Param{{paramUser, valNaoina}, {paramOrg, valSomething}}, true},
		{pathFeeds, pathFeeds, nil, true},
		{pathNotifications, pathNotifications, nil, true},
		{"/repos/naoina/denco/notifications", pathReposNotifications, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/notifications/threads/1", pathNotificationThreads, []denco.Param{{paramID, "1"}}, true},
		{"/notifications/threads/2/subscription", pathNotificationThreadSub, []denco.Param{{paramID, "2"}}, true},
		{"/repos/naoina/denco/stargazers", pathReposStargazers, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/users/naoina/starred", pathUsersStarred, []denco.Param{{paramUser, valNaoina}}, true},
		{pathUserStarred, pathUserStarred, nil, true},
		{"/user/starred/naoina/denco", pathUserStarredOwnerRepo, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/subscribers", pathReposSubscribers, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/users/naoina/subscriptions", pathUsersSubscriptions, []denco.Param{{paramUser, valNaoina}}, true},
		{pathUserSubscriptions, pathUserSubscriptions, nil, true},
		{"/repos/naoina/denco/subscription", pathReposSubscription, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/user/subscriptions/naoina/denco", pathUserSubscriptionsOwnerRepo, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/users/naoina/gists", pathUsersGists, []denco.Param{{paramUser, valNaoina}}, true},
		{pathGists, pathGists, nil, true},
		{"/gists/1", pathGistsID, []denco.Param{{paramID, "1"}}, true},
		{"/gists/2/star", pathGistsIDStar, []denco.Param{{paramID, "2"}}, true},
		{"/repos/naoina/denco/git/blobs/03c3bbc7f0d12268b9ca53d4fbfd8dc5ae5697b9", pathReposGitBlobs, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramSHA, valSHA1}}, true},
		{"/repos/naoina/denco/git/commits/03c3bbc7f0d12268b9ca53d4fbfd8dc5ae5697b9", pathReposGitCommits, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramSHA, valSHA1}}, true},
		{"/repos/naoina/denco/git/refs", pathReposGitRefs, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/git/tags/03c3bbc7f0d12268b9ca53d4fbfd8dc5ae5697b9", pathReposGitTags, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramSHA, valSHA1}}, true},
		{"/repos/naoina/denco/git/trees/03c3bbc7f0d12268b9ca53d4fbfd8dc5ae5697b9", pathReposGitTrees, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramSHA, valSHA1}}, true},
		{pathIssues, pathIssues, nil, true},
		{pathUserIssues, pathUserIssues, nil, true},
		{"/orgs/something/issues", pathOrgsIssues, []denco.Param{{paramOrg, valSomething}}, true},
		{"/repos/naoina/denco/issues", pathReposIssues, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/issues/1", pathReposIssue, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramNumber, "1"}}, true},
		{"/repos/naoina/denco/assignees", pathReposAssignees, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/assignees/foo", pathReposAssignee, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {"assignee", valFoo}}, true},
		{"/repos/naoina/denco/issues/1/comments", pathReposIssueComments, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramNumber, "1"}}, true},
		{"/repos/naoina/denco/issues/1/events", pathReposIssueEvents, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramNumber, "1"}}, true},
		{"/repos/naoina/denco/labels", pathReposLabels, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/labels/bug", pathReposLabel, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {"name", "bug"}}, true},
		{"/repos/naoina/denco/issues/1/labels", pathReposIssueLabels, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramNumber, "1"}}, true},
		{"/repos/naoina/denco/milestones/1/labels", pathReposMilestoneLabels, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramNumber, "1"}}, true},
		{"/repos/naoina/denco/milestones", pathReposMilestones, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/milestones/1", pathReposMilestone, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramNumber, "1"}}, true},
		{pathEmojis, pathEmojis, nil, true},
		{pathGitignoreTemplates, pathGitignoreTemplates, nil, true},
		{"/gitignore/templates/Go", pathGitignoreTemplate, []denco.Param{{"name", "Go"}}, true},
		{pathMeta, pathMeta, nil, true},
		{pathRateLimit, pathRateLimit, nil, true},
		{"/users/naoina/orgs", pathUsersOrgs, []denco.Param{{paramUser, valNaoina}}, true},
		{pathUserOrgs, pathUserOrgs, nil, true},
		{"/orgs/something", pathOrgsOrg, []denco.Param{{paramOrg, valSomething}}, true},
		{"/orgs/something/members", pathOrgsMembers, []denco.Param{{paramOrg, valSomething}}, true},
		{"/orgs/something/members/naoina", pathOrgsMember, []denco.Param{{paramOrg, valSomething}, {paramUser, valNaoina}}, true},
		{"/orgs/something/public_members", pathOrgsPublicMembers, []denco.Param{{paramOrg, valSomething}}, true},
		{"/orgs/something/public_members/naoina", pathOrgsPublicMember, []denco.Param{{paramOrg, valSomething}, {paramUser, valNaoina}}, true},
		{"/orgs/something/teams", pathOrgsTeams, []denco.Param{{paramOrg, valSomething}}, true},
		{"/teams/1", pathTeamsID, []denco.Param{{paramID, "1"}}, true},
		{"/teams/2/members", pathTeamsMembers, []denco.Param{{paramID, "2"}}, true},
		{"/teams/3/members/naoina", pathTeamsMember, []denco.Param{{paramID, "3"}, {paramUser, valNaoina}}, true},
		{"/teams/4/repos", pathTeamsRepos, []denco.Param{{paramID, "4"}}, true},
		{"/teams/5/repos/naoina/denco", pathTeamsRepo, []denco.Param{{paramID, "5"}, {paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{pathUserTeams, pathUserTeams, nil, true},
		{"/repos/naoina/denco/pulls", pathReposPulls, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/pulls/1", pathReposPull, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramNumber, "1"}}, true},
		{"/repos/naoina/denco/pulls/1/commits", pathReposPullCommits, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramNumber, "1"}}, true},
		{"/repos/naoina/denco/pulls/1/files", pathReposPullFiles, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramNumber, "1"}}, true},
		{"/repos/naoina/denco/pulls/1/merge", pathReposPullMerge, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramNumber, "1"}}, true},
		{"/repos/naoina/denco/pulls/1/comments", pathReposPullComments, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramNumber, "1"}}, true},
		{pathUserRepos, pathUserRepos, nil, true},
		{"/users/naoina/repos", pathUsersRepos, []denco.Param{{paramUser, valNaoina}}, true},
		{"/orgs/something/repos", pathOrgsRepos, []denco.Param{{paramOrg, valSomething}}, true},
		{pathRepositories, pathRepositories, nil, true},
		{"/repos/naoina/denco", pathRepoOwnerRepo, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/contributors", pathReposContributors, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/languages", pathReposLanguages, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/teams", pathReposTeams, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/tags", pathReposTags, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/branches", pathReposBranches, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/branches/master", pathReposBranch, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {"branch", "master"}}, true},
		{"/repos/naoina/denco/collaborators", pathReposCollaborators, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/collaborators/something", pathReposCollaborator, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramUser, valSomething}}, true},
		{"/repos/naoina/denco/comments", pathReposComments, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/commits/03c3bbc7f0d12268b9ca53d4fbfd8dc5ae5697b9/comments", pathReposCommitsSHAComments, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramSHA, valSHA1}}, true},
		{"/repos/naoina/denco/comments/1", pathReposComment, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramID, "1"}}, true},
		{"/repos/naoina/denco/commits", pathReposCommits, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/commits/03c3bbc7f0d12268b9ca53d4fbfd8dc5ae5697b9", pathReposCommit, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramSHA, valSHA1}}, true},
		{"/repos/naoina/denco/readme", pathReposReadme, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/keys", pathReposKeys, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/keys/1", pathReposKey, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramID, "1"}}, true},
		{"/repos/naoina/denco/downloads", pathReposDownloads, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/downloads/2", pathReposDownload, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramID, "2"}}, true},
		{"/repos/naoina/denco/forks", pathReposForks, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/hooks", pathReposHooks, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/hooks/2", pathReposHook, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramID, "2"}}, true},
		{"/repos/naoina/denco/releases", pathReposReleases, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/releases/1", pathReposRelease, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramID, "1"}}, true},
		{"/repos/naoina/denco/releases/1/assets", pathReposReleaseAssets, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {paramID, "1"}}, true},
		{"/repos/naoina/denco/stats/contributors", pathReposStatsContributors, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/stats/commit_activity", pathReposStatsCommitActivity, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/stats/code_frequency", pathReposStatsCodeFrequency, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/stats/participation", pathReposStatsParticipation, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/stats/punch_card", pathReposStatsPunchCard, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}}, true},
		{"/repos/naoina/denco/statuses/master", pathReposStatuses, []denco.Param{{paramOwner, valNaoina}, {paramRepo, valDenco}, {"ref", "master"}}, true},
		{pathSearchRepositories, pathSearchRepositories, nil, true},
		{pathSearchCode, pathSearchCode, nil, true},
		{pathSearchIssues, pathSearchIssues, nil, true},
		{pathSearchUsers, pathSearchUsers, nil, true},
		{"/legacy/issues/search/naoina/denco/closed/test", pathLegacyIssuesSearch, []denco.Param{{paramOwner, valNaoina}, {"repository", valDenco}, {"state", "closed"}, {paramKeyword, valTest}}, true},
		{"/legacy/repos/search/test", pathLegacyReposSearch, []denco.Param{{paramKeyword, valTest}}, true},
		{"/legacy/user/search/test", pathLegacyUserSearch, []denco.Param{{paramKeyword, valTest}}, true},
		{"/legacy/user/email/naoina@kuune.org", pathLegacyUserEmail, []denco.Param{{"email", "naoina@kuune.org"}}, true},
		{"/users/naoina", pathUsersUser, []denco.Param{{paramUser, valNaoina}}, true},
		{pathUser, pathUser, nil, true},
		{pathUsers, pathUsers, nil, true},
		{pathUserEmails, pathUserEmails, nil, true},
		{"/users/naoina/followers", pathUsersFollowers, []denco.Param{{paramUser, valNaoina}}, true},
		{pathUserFollowers, pathUserFollowers, nil, true},
		{"/users/naoina/following", pathUsersFollowing, []denco.Param{{paramUser, valNaoina}}, true},
		{pathUserFollowing, pathUserFollowing, nil, true},
		{"/user/following/naoina", pathUserFollowingUser, []denco.Param{{paramUser, valNaoina}}, true},
		{"/users/naoina/following/target", pathUsersFollowingTarget, []denco.Param{{paramUser, valNaoina}, {"target_user", "target"}}, true},
		{"/users/naoina/keys", pathUsersKeys, []denco.Param{{paramUser, valNaoina}}, true},
		{pathUserKeys, pathUserKeys, nil, true},
		{"/user/keys/1", pathUserKey, []denco.Param{{paramID, "1"}}, true},
		{"/people/me", pathPeopleUserID, []denco.Param{{paramUserID, valMe}}, true},
		{pathPeople, pathPeople, nil, true},
		{"/activities/foo/people/vault", pathActivitiesPeople, []denco.Param{{paramActivityID, valFoo}, {paramCollection, valVault}}, true},
		{"/people/me/people/vault", pathPeoplePeople, []denco.Param{{paramUserID, valMe}, {paramCollection, valVault}}, true},
		{"/people/me/openIdConnect", pathPeopleOpenIDConnect, []denco.Param{{paramUserID, valMe}}, true},
		{"/people/me/activities/vault", pathPeopleActivities, []denco.Param{{paramUserID, valMe}, {paramCollection, valVault}}, true},
		{"/activities/foo", pathActivitiesActivityID, []denco.Param{{paramActivityID, valFoo}}, true},
		{pathActivities, pathActivities, nil, true},
		{"/activities/foo/comments", pathActivitiesComments, []denco.Param{{paramActivityID, valFoo}}, true},
		{"/comments/hoge", pathCommentsCommentID, []denco.Param{{"commentId", "hoge"}}, true},
		{"/people/me/moments/vault", pathPeopleMoments, []denco.Param{{paramUserID, valMe}, {paramCollection, valVault}}, true},
	}
	runLookupTest(t, realURIs, testcases)
}

func TestRouter_Build(t *testing.T) {
	// test for duplicate name of path parameters.
	func() {
		r := denco.New()
		require.Errorf(t,
			r.Build([]denco.Record{
				{"/:user/:id/:id", testRoute0},
				{"/:user/:user/:id", testRoute0},
			}),
			"no error returned by duplicate name of path parameters",
		)
	}()
}

func TestRouter_Build_withoutSizeHint(t *testing.T) {
	for _, v := range []struct {
		keys     []string
		sizeHint int
	}{
		{[]string{pathUser}, 0},
		{[]string{pathUserID}, 1},
		{[]string{"/user/:id/post"}, 1},
		{[]string{"/user/:id/post:validate"}, 2},
		{[]string{pathUserIDGroup}, 2},
		{[]string{pathUserIDPostCID}, 2},
		{[]string{pathUserIDPostCID, "/admin/:id/post/:cid"}, 2},
		{[]string{pathUserID, "/admin/:id/post/:cid"}, 2},
		{[]string{pathUserIDPostCID, "/admin/:id/post/:cid/:type"}, 3},
	} {
		r := denco.New()
		actual := r.SizeHint
		expected := -1

		assert.EqualTf(t, expected, actual, `before Build; Router.SizeHint => (%[1]T=%#[1]v); want (%[2]T=%#[2]v)`, actual, expected)
		records := make([]denco.Record, len(v.keys))
		for i, k := range v.keys {
			records[i] = denco.Record{Key: k, Value: paramValue}
		}
		require.NoError(t, r.Build(records))
		actual = r.SizeHint
		expected = v.sizeHint
		assert.EqualTf(t, expected, actual, `Router.Build(%#v); Router.SizeHint => (%[2]T=%#[2]v); want (%[3]T=%#[3]v)`, records, actual, expected)
	}
}

func TestRouter_Build_withSizeHint(t *testing.T) {
	for _, v := range []struct {
		key      string
		sizeHint int
		expect   int
	}{
		{pathUser, 0, 0},
		{pathUser, 1, 1},
		{pathUser, 2, 2},
		{pathUserID, 3, 3},
		{pathUserIDGroup, 0, 0},
		{pathUserIDGroup, 1, 1},
		{"/user/:id/:group:validate", 1, 1},
	} {
		r := denco.New()
		r.SizeHint = v.sizeHint
		records := []denco.Record{
			{v.key, paramValue},
		}
		require.NoError(t, r.Build(records))
		actual := r.SizeHint
		expected := v.expect
		assert.EqualTf(t, expected, actual, `Router.Build(%#v); Router.SizeHint => (%[2]T=%#[2]v); want (%[3]T=%#[3]v)`, records, actual, expected)
	}
}

func TestParams_Get(t *testing.T) {
	params := denco.Params([]denco.Param{
		{paramName1, "value1"},
		{"name2", "value2"},
		{"name3", "value3"},
		{paramName1, "value4"},
	})
	for _, v := range []struct{ value, expected string }{
		{paramName1, "value1"},
		{"name2", "value2"},
		{"name3", "value3"},
		{"name4", ""},
	} {
		actual := params.Get(v.value)
		expected := v.expected
		assert.EqualT(t, expected, actual, "Params.Get(%q) => %#v, want %#v", v.value, actual, expected)
	}
}
