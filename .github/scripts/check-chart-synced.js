const { Octokit } = require("@octokit/rest");
const github = require('@actions/github');

const token = process.env.GITHUB_TOKEN;
const syncDirectories = process.env.SYNC_DIRECTORIES.split(',').map(dir => dir.trim());
const octokit = new Octokit({ auth: token });

const context = github.context;
const { owner, repo } = context.repo;
const pull_number = context.payload.pull_request.number;
const labelName = 'require-chart-sync';
const bypassLabel = 'skip-chart-sync';
const chartsRepo = 'odigos-charts';  // The repository name for Odigos-charts

async function checkPR() {
    try {
        // Get the labels on the PR
        const { data: prLabels } = await octokit.issues.listLabelsOnIssue({
            owner,
            repo,
            issue_number: pull_number
        });

        // Skip check if bypass label is present
        if (prLabels.some(label => label.name === bypassLabel)) {
            console.log(`Bypass label '${bypassLabel}' found. Skipping check.`);
            return { needSync: 'false' };
        }

        // Get list of files changed in the PR
        const { data: files } = await octokit.pulls.listFiles({
            owner,
            repo,
            pull_number
        });

        // Check if any file in the specified directories is changed
        const configFilesChanged = files.some(file =>
            syncDirectories.some(dir => file.filename.startsWith(dir))
        );

        if (!configFilesChanged) {
            console.log("No changes in specified directories. Exiting.");
            return { needSync: 'false' };
        }

        // Get the comments in the PR
        const { data: comments } = await octokit.issues.listComments({
            owner,
            repo,
            issue_number: pull_number
        });

        // Check if any comment contains a reference to Odigos-charts repository
        const referenceFoundInComments = comments.some(comment => /odigos-charts/.test(comment.body));

        // Check if PR description references a PR from Odigos-charts repository
        const referenceFoundInDescription = context.payload.pull_request.body.includes(`github.com/${owner}/${chartsRepo}/pull/`);

        if (!referenceFoundInComments && !referenceFoundInDescription) {
            // If no reference found, create a comment and block the PR
            await octokit.issues.createComment({
                owner,
                repo,
                issue_number: pull_number,
                body: "**This PR includes changes in specified directories.** Please ensure that changes are synced with the [odigos-charts](https://github.com/odigos-io/odigos-charts) repository. If the changes are not related to Odigos-charts, please add the label 'skip-chart-sync'. If the changes are related to Odigos-charts, please add a reference to the Odigos-charts PR in the description or comments."
            });

            // Add a label to block the PR
            await octokit.issues.addLabels({
                owner,
                repo,
                issue_number: pull_number,
                labels: [labelName]
            });

            return { needSync: 'true' };
        } else {
            // Remove the label if it exists and the check passes
            if (prLabels.some(label => label.name === labelName)) {
                await octokit.issues.removeLabel({
                    owner,
                    repo,
                    issue_number: pull_number,
                    name: labelName
                });
            }

            console.log("Reference to Odigos-charts found. PR can proceed.");
            return { needSync: 'false' };
        }
    } catch (error) {
        console.error("Error checking PR:", error);
        return { needSync: 'true' };
    }
}

checkPR().then(output => {
    // Set the output for the GitHub Action
    console.log(`::set-output name=need_sync::${output.needSync}`);
});
