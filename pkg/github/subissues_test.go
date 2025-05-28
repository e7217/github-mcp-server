package github

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/github/github-mcp-server/pkg/translations"
	"github.com/google/go-github/v72/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ListSubIssues(t *testing.T) {
	// Verify tool definition
	mockClient := github.NewClient(nil)
	tool, _ := ListSubIssues(stubGetClientFn(mockClient), translations.NullTranslationHelper)

	assert.Equal(t, "list_sub_issues", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "owner")
	assert.Contains(t, tool.InputSchema.Properties, "repo")
	assert.Contains(t, tool.InputSchema.Properties, "issue_number")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"owner", "repo", "issue_number"})

	// Mock sub-issues data
	mockSubIssues := []*github.Issue{
		{
			Number:  github.Ptr(101),
			Title:   github.Ptr("Sub-issue 1"),
			Body:    github.Ptr("First sub-issue"),
			State:   github.Ptr("open"),
			HTMLURL: github.Ptr("https://github.com/owner/repo/issues/101"),
		},
		{
			Number:  github.Ptr(102),
			Title:   github.Ptr("Sub-issue 2"),
			Body:    github.Ptr("Second sub-issue"),
			State:   github.Ptr("closed"),
			HTMLURL: github.Ptr("https://github.com/owner/repo/issues/102"),
		},
	}

	tests := []struct {
		name              string
		mockedClient      *http.Client
		requestArgs       map[string]interface{}
		expectError       bool
		expectedSubIssues []*github.Issue
		expectedErrMsg    string
	}{
		{
			name: "successful sub-issues listing",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.EndpointPattern{
						Pattern: "/repos/owner/repo/issues/42/sub_issues",
						Method:  "GET",
					},
					mockResponse(t, http.StatusOK, mockSubIssues),
				),
			),
			requestArgs: map[string]interface{}{
				"owner":        "owner",
				"repo":         "repo",
				"issue_number": float64(42),
			},
			expectError:       false,
			expectedSubIssues: mockSubIssues,
		},
		{
			name: "issue not found",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.EndpointPattern{
						Pattern: "/repos/owner/repo/issues/999/sub_issues",
						Method:  "GET",
					},
					mockResponse(t, http.StatusNotFound, `{"message": "Issue not found"}`),
				),
			),
			requestArgs: map[string]interface{}{
				"owner":        "owner",
				"repo":         "repo",
				"issue_number": float64(999),
			},
			expectError:    true,
			expectedErrMsg: "failed to list sub-issues",
		},
		{
			name: "missing required parameter",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.EndpointPattern{
						Pattern: "/repos/owner/repo/issues/42/sub_issues",
						Method:  "GET",
					},
					mockResponse(t, http.StatusOK, mockSubIssues),
				),
			),
			requestArgs: map[string]interface{}{
				"owner": "owner",
				"repo":  "repo",
				// missing issue_number
			},
			expectError:    false, // This will be handled as a tool result error
			expectedErrMsg: "missing required parameter: issue_number",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup client with mock
			client := github.NewClient(tc.mockedClient)
			_, handler := ListSubIssues(stubGetClientFn(client), translations.NullTranslationHelper)

			// Create call request
			request := createMCPRequest(tc.requestArgs)

			// Call handler
			result, err := handler(context.Background(), request)

			// Verify results
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
				return
			}

			require.NoError(t, err)

			// Check for tool result error
			if tc.expectedErrMsg != "" {
				textContent := getTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedErrMsg)
				return
			}

			textContent := getTextResult(t, result)

			// Unmarshal and verify the result
			var returnedSubIssues []*github.Issue
			err = json.Unmarshal([]byte(textContent.Text), &returnedSubIssues)
			require.NoError(t, err)
			assert.Len(t, returnedSubIssues, len(tc.expectedSubIssues))

			for i, expected := range tc.expectedSubIssues {
				assert.Equal(t, *expected.Number, *returnedSubIssues[i].Number)
				assert.Equal(t, *expected.Title, *returnedSubIssues[i].Title)
			}
		})
	}
}

func Test_AddSubIssue(t *testing.T) {
	// Verify tool definition
	mockClient := github.NewClient(nil)
	tool, _ := AddSubIssue(stubGetClientFn(mockClient), translations.NullTranslationHelper)

	assert.Equal(t, "add_sub_issue", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "owner")
	assert.Contains(t, tool.InputSchema.Properties, "repo")
	assert.Contains(t, tool.InputSchema.Properties, "issue_number")
	assert.Contains(t, tool.InputSchema.Properties, "sub_issue_id")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"owner", "repo", "issue_number", "sub_issue_id"})

	// Mock updated issue with sub-issue added
	mockUpdatedIssue := &github.Issue{
		Number:  github.Ptr(42),
		Title:   github.Ptr("Parent Issue"),
		Body:    github.Ptr("This issue now has sub-issues"),
		State:   github.Ptr("open"),
		HTMLURL: github.Ptr("https://github.com/owner/repo/issues/42"),
	}

	tests := []struct {
		name           string
		mockedClient   *http.Client
		requestArgs    map[string]interface{}
		expectError    bool
		expectedIssue  *github.Issue
		expectedErrMsg string
	}{
		{
			name: "successful sub-issue addition",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.EndpointPattern{
						Pattern: "/repos/owner/repo/issues/42/sub_issues",
						Method:  "PUT",
					},
					mockResponse(t, http.StatusOK, mockUpdatedIssue),
				),
			),
			requestArgs: map[string]interface{}{
				"owner":        "owner",
				"repo":         "repo",
				"issue_number": float64(42),
				"sub_issue_id": float64(101),
			},
			expectError:   false,
			expectedIssue: mockUpdatedIssue,
		},
		{
			name: "sub-issue already exists",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.EndpointPattern{
						Pattern: "/repos/owner/repo/issues/42/sub_issues",
						Method:  "PUT",
					},
					mockResponse(t, http.StatusUnprocessableEntity, `{"message": "Sub-issue already exists"}`),
				),
			),
			requestArgs: map[string]interface{}{
				"owner":        "owner",
				"repo":         "repo",
				"issue_number": float64(42),
				"sub_issue_id": float64(101),
			},
			expectError:    false,
			expectedErrMsg: "failed to add sub-issue: Sub-issue already exists",
		},
		{
			name: "missing required parameter",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.EndpointPattern{
						Pattern: "/repos/owner/repo/issues/42/sub_issues",
						Method:  "PUT",
					},
					mockResponse(t, http.StatusOK, mockUpdatedIssue),
				),
			),
			requestArgs: map[string]interface{}{
				"owner":        "owner",
				"repo":         "repo",
				"issue_number": float64(42),
				// missing sub_issue_id
			},
			expectError:    false,
			expectedErrMsg: "missing required parameter: sub_issue_id",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup client with mock
			client := github.NewClient(tc.mockedClient)
			_, handler := AddSubIssue(stubGetClientFn(client), translations.NullTranslationHelper)

			// Create call request
			request := createMCPRequest(tc.requestArgs)

			// Call handler
			result, err := handler(context.Background(), request)

			// Verify results
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
				return
			}

			require.NoError(t, err)

			// Check for tool result error
			if tc.expectedErrMsg != "" {
				textContent := getTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedErrMsg)
				return
			}

			textContent := getTextResult(t, result)

			// Unmarshal and verify the result
			var returnedIssue github.Issue
			err = json.Unmarshal([]byte(textContent.Text), &returnedIssue)
			require.NoError(t, err)
			assert.Equal(t, *tc.expectedIssue.Number, *returnedIssue.Number)
			assert.Equal(t, *tc.expectedIssue.Title, *returnedIssue.Title)
		})
	}
}

func Test_RemoveSubIssue(t *testing.T) {
	// Verify tool definition
	mockClient := github.NewClient(nil)
	tool, _ := RemoveSubIssue(stubGetClientFn(mockClient), translations.NullTranslationHelper)

	assert.Equal(t, "remove_sub_issue", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "owner")
	assert.Contains(t, tool.InputSchema.Properties, "repo")
	assert.Contains(t, tool.InputSchema.Properties, "issue_number")
	assert.Contains(t, tool.InputSchema.Properties, "sub_issue_id")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"owner", "repo", "issue_number", "sub_issue_id"})

	// Mock updated issue with sub-issue removed
	mockUpdatedIssue := &github.Issue{
		Number:  github.Ptr(42),
		Title:   github.Ptr("Parent Issue"),
		Body:    github.Ptr("This issue had a sub-issue removed"),
		State:   github.Ptr("open"),
		HTMLURL: github.Ptr("https://github.com/owner/repo/issues/42"),
	}

	tests := []struct {
		name           string
		mockedClient   *http.Client
		requestArgs    map[string]interface{}
		expectError    bool
		expectedIssue  *github.Issue
		expectedErrMsg string
	}{
		{
			name: "successful sub-issue removal",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.EndpointPattern{
						Pattern: "/repos/owner/repo/issues/42/sub_issues",
						Method:  "DELETE",
					},
					mockResponse(t, http.StatusOK, mockUpdatedIssue),
				),
			),
			requestArgs: map[string]interface{}{
				"owner":        "owner",
				"repo":         "repo",
				"issue_number": float64(42),
				"sub_issue_id": float64(101),
			},
			expectError:   false,
			expectedIssue: mockUpdatedIssue,
		},
		{
			name: "sub-issue not found",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.EndpointPattern{
						Pattern: "/repos/owner/repo/issues/42/sub_issues",
						Method:  "DELETE",
					},
					mockResponse(t, http.StatusNotFound, `{"message": "Sub-issue not found"}`),
				),
			),
			requestArgs: map[string]interface{}{
				"owner":        "owner",
				"repo":         "repo",
				"issue_number": float64(42),
				"sub_issue_id": float64(999),
			},
			expectError:    false,
			expectedErrMsg: "failed to remove sub-issue: Sub-issue not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup client with mock
			client := github.NewClient(tc.mockedClient)
			_, handler := RemoveSubIssue(stubGetClientFn(client), translations.NullTranslationHelper)

			// Create call request
			request := createMCPRequest(tc.requestArgs)

			// Call handler
			result, err := handler(context.Background(), request)

			// Verify results
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
				return
			}

			require.NoError(t, err)

			// Check for tool result error
			if tc.expectedErrMsg != "" {
				textContent := getTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedErrMsg)
				return
			}

			textContent := getTextResult(t, result)

			// Unmarshal and verify the result
			var returnedIssue github.Issue
			err = json.Unmarshal([]byte(textContent.Text), &returnedIssue)
			require.NoError(t, err)
			assert.Equal(t, *tc.expectedIssue.Number, *returnedIssue.Number)
			assert.Equal(t, *tc.expectedIssue.Title, *returnedIssue.Title)
		})
	}
}

func Test_ReprioritizeSubIssue(t *testing.T) {
	// Verify tool definition
	mockClient := github.NewClient(nil)
	tool, _ := ReprioritizeSubIssue(stubGetClientFn(mockClient), translations.NullTranslationHelper)

	assert.Equal(t, "reprioritize_sub_issue", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "owner")
	assert.Contains(t, tool.InputSchema.Properties, "repo")
	assert.Contains(t, tool.InputSchema.Properties, "issue_number")
	assert.Contains(t, tool.InputSchema.Properties, "sub_issue_id")
	assert.Contains(t, tool.InputSchema.Properties, "after_id")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"owner", "repo", "issue_number", "sub_issue_id"})

	// Mock updated issue with reprioritized sub-issues
	mockUpdatedIssue := &github.Issue{
		Number:  github.Ptr(42),
		Title:   github.Ptr("Parent Issue"),
		Body:    github.Ptr("This issue has reprioritized sub-issues"),
		State:   github.Ptr("open"),
		HTMLURL: github.Ptr("https://github.com/owner/repo/issues/42"),
	}

	tests := []struct {
		name           string
		mockedClient   *http.Client
		requestArgs    map[string]interface{}
		expectError    bool
		expectedIssue  *github.Issue
		expectedErrMsg string
	}{
		{
			name: "successful sub-issue reprioritization with after_id",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.EndpointPattern{
						Pattern: "/repos/owner/repo/issues/42/sub_issues/priority",
						Method:  "PATCH",
					},
					mockResponse(t, http.StatusOK, mockUpdatedIssue),
				),
			),
			requestArgs: map[string]interface{}{
				"owner":        "owner",
				"repo":         "repo",
				"issue_number": float64(42),
				"sub_issue_id": float64(101),
				"after_id":     float64(102),
			},
			expectError:   false,
			expectedIssue: mockUpdatedIssue,
		},
		{
			name: "successful sub-issue reprioritization without after_id",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.EndpointPattern{
						Pattern: "/repos/owner/repo/issues/42/sub_issues/priority",
						Method:  "PATCH",
					},
					mockResponse(t, http.StatusOK, mockUpdatedIssue),
				),
			),
			requestArgs: map[string]interface{}{
				"owner":        "owner",
				"repo":         "repo",
				"issue_number": float64(42),
				"sub_issue_id": float64(101),
			},
			expectError:   false,
			expectedIssue: mockUpdatedIssue,
		},
		{
			name: "sub-issue not found for reprioritization",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.EndpointPattern{
						Pattern: "/repos/owner/repo/issues/42/sub_issues/priority",
						Method:  "PATCH",
					},
					mockResponse(t, http.StatusNotFound, `{"message": "Sub-issue not found"}`),
				),
			),
			requestArgs: map[string]interface{}{
				"owner":        "owner",
				"repo":         "repo",
				"issue_number": float64(42),
				"sub_issue_id": float64(999),
			},
			expectError:    false,
			expectedErrMsg: "failed to reprioritize sub-issue: Sub-issue not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup client with mock
			client := github.NewClient(tc.mockedClient)
			_, handler := ReprioritizeSubIssue(stubGetClientFn(client), translations.NullTranslationHelper)

			// Create call request
			request := createMCPRequest(tc.requestArgs)

			// Call handler
			result, err := handler(context.Background(), request)

			// Verify results
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
				return
			}

			require.NoError(t, err)

			// Check for tool result error
			if tc.expectedErrMsg != "" {
				textContent := getTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedErrMsg)
				return
			}

			textContent := getTextResult(t, result)

			// Unmarshal and verify the result
			var returnedIssue github.Issue
			err = json.Unmarshal([]byte(textContent.Text), &returnedIssue)
			require.NoError(t, err)
			assert.Equal(t, *tc.expectedIssue.Number, *returnedIssue.Number)
			assert.Equal(t, *tc.expectedIssue.Title, *returnedIssue.Title)
		})
	}
}
