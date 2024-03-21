/*
 * Copyright 2022 Aspect Build Systems, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

 package main

 import (
	 "context"
	 "fmt"
	 "path/filepath"
	 "strings"
 
	 "github.com/manifoldco/promptui"
	 "github.com/spf13/cobra"
 
	 rootFlags "github.com/aspect-build/silo/cli/core/pkg/aspect/root/flags"
	 "github.com/aspect-build/silo/cli/core/pkg/bazel"
	 "github.com/aspect-build/silo/cli/core/pkg/ioutils"
	 "github.com/slack-go/slack"
 )
 
 const (
	 slackAppClientId = "1234566.1234567"
	 tokenLambdaUrl   = "https://identifier-here.lambda-url.us-west-2.on.aws/"
 )
 
 type Support struct {
	 ioutils.Streams
 }
 
 func New(streams ioutils.Streams) *Support {
	 return &Support{
		 Streams: streams,
	 }
 }
 
 func (runner *Support) Run(ctx context.Context, cmd *cobra.Command, args []string) error {
	 fmt.Println(`To provide support, this plugin posts a message to Slack.
 So we'll need to authenticate to Slack first.
 We'll open your browser and navigate to Slack's authorization page.
 You'll be asked to permit our Aspect Bazel Support app to post messages on your behalf.`)
	 applyFixPrompt := promptui.Prompt{
		 Label:     "Ready to authenticate with Slack",
		 IsConfirm: true,
	 }
	 _, reject := applyFixPrompt.Run()
	 if reject != nil {
		 fmt.Println("Okay then, sorry we weren't able to handle your support request.")
		 return nil
	 }
 
	 // TODO(alex): persist the token somewhere secure, use it again in the next invocation
	 // also verify that the stored token is still valid, if not discard and do oauth again.
	 token, err := HandleOpenIDFlow(slackAppClientId, tokenLambdaUrl)
	 if err != nil {
		 return fmt.Errorf("cannot authenticate to slack: %w", err)
	 }
	 // TODO(alex): look into a store of previous invocations for relevant logs
	 bzl := bazel.WorkspaceFromWd
 
	 var out strings.Builder
	 streams := ioutils.Streams{Stdout: &out, Stderr: nil}
	 if err := bzl.RunCommand(streams, nil, "info", "output_base"); err != nil {
		 return fmt.Errorf("unable to locate output_base: %w", err)
	 }
	 outputBase := strings.TrimSpace(out.String())
 
	 api := slack.New(token)
 
	 _, timestamp, _, err := api.SendMessageContext(ctx, "C0463333N3", slack.MsgOptionBlocks(
		 slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", ":pray: *Build Result*:", false, false), nil, nil),
		 slack.NewContextBlock("", slack.NewTextBlockObject("plain_text", "submitted from the Aspect CLI", false, false)),
	 ))
	 if err != nil {
		 return fmt.Errorf("cannot post to slack: %w", err)
	 }
	 _, err = api.UploadFileContext(ctx, slack.FileUploadParameters{
		 File:            filepath.Join(outputBase, "command.log"),
		 Filetype:        "txt",
		 ThreadTimestamp: timestamp,
		 Channels:        []string{"C0463333N3"},
	 })
	 if err != nil {
		 return fmt.Errorf("failed to upload command.log: %w", err)
	 }

	 return nil
 }
 