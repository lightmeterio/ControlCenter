# Development Flow (draft 1)

The SCM tool we use is Git and we adopt a development flow modeled after the widely used [Gitflow](https://www.atlassian.com/git/tutorials/comparing-workflows/gitflow-workflow). There are some differences though, explained below.

We use Gitlab as a tool for managing the code and the process. Our main repository is https://gitlab.com/lightmeter/controlcenter.

## New features

Every new feature stars by forking `develop` into a branch called `feature/<issue_number>_some_descriptive_name`, where `<issue_number>` is the gitlab issue number related to code to be developed.

Once the developer feels confident with their code, they'll create a merge request (MR) based on `develop` of such feature branch.

Choosing one or more reviewers is up to each individual request. A rule of thumb is assign someone familiar with the part of the code being changed.
Or just talk to other team members to find someone to review your MR. Don't be too strict on it :-)

Once the MR has been approved, the developer is responsible for merging it.

If possible, cleaning up the git history of the change is appreciated, squashing, ammending commits to prevent lots of "fixup" commits. It should be done mefore the merge and agreed with the reviewers.

It's also important to rebase `develop` on the feature branch, instead of merge. This is in order to have a linear history on develop and prevent

### Language translation on new features

It's important that requests for new features not to include translated strings, as they have to be done via our [translation platform](https://translate.lightmeter.io).

When translations are done via such interface, a new merge request is created based on `master`. This happend because translations must be made on stable versions, and must not be done on development ones. See the file `TRANSLATION.md` for more information.

## Landing to master

As many features might've been landed to `develop`, the process of landing then on master requires a bit more steps, but it's quite mechanical.
It consists on cherry-picking only the commits of a given feature to be merged on `master`.

### Method 1

As develop will contain many merge commits and commits from multiple features interleaved, some cleaning will be needed to make it "cherry-pickeable".

Let's suppose the feature I want to merge on master is between the commits `aaaaaaa` (with it included) and `bbbbbbb` (with other unwanted commits in between):

```sh
master: git checkout -b temp-clean-feature-history bbbbbbb
temp-clean-feature-history: git rebase -i aaaaaaa~1
```

In the rebase editor, remove any commits not related to the feature you want to merge on master.

Save and leave the editor. It's possible that you have to manually fix rebase issues. More information about it in the `git rebase` documentation.

After finishing the rebasing, you should have a linear history with all the commits related to the feature in the branch `temp-clean-feature-history`.

Now move back to the master branch and create a feature branch and cherry-pick such commits:

```sh
temp-clean-feature-history: git checkout master
master: git checkout -b feature/XXX_my_super_feature
feature/XXX_my_super_feature: git cherry-pick -x aaaaaaa~1..temp-clean-feature-history
```

***NOTE***: notice the usage of `aaaaaaa~1`. It happens because `cherry-pick` the interval is open on its start, meaning that we are need to include `aaaaaa` to the merge request.

Then delete the temporary branch:

```sh
feature/XXX_my_super_feature: git branch -D temp-clean-feature-history
```

### Method 2

Alternatively, in case you know exactly which commits to cherry-pick from develop, you can simplify the process.

Let's suppose you want to cherry-pick the commits `aaaaaa` and `bbbbbb` from develop on a MR to master. Do as following:

```sh
master: git checkout -b feature/XXX_my_super_feature
feature/XXX_my_super_feature: git cherry-pick -x aaaaaa bbbbbb
```

### Create a Merge request

Finally create a MR from your branch to `master`. In the example, I'm using the [lab](https://github.com/zaquestion/lab) tool, but you can use use any other tool or the Gitlab UI:

```sh
feature/XXX_my_super_feature: lab mr create origin master -s
```

Merge requests made on master ***must*** be squashed upon merging. It's necessary in order to make reverting changes easier.

Did you see? It did not hurt, dit it? :-)

### When stuff gets done

Some issue/story/fix is considered done only when it's successfully landed to master.

## Hotfixes

Fix requests start by forking `master` or `develop`, depending on where it should be applied. The branch name should be `fix/<issue_number>_some_descriptive_name`.

Similar to feature request, such hotfixes should go through the code reviewing process. If it's merged to `master`, it should be available for the next release candidate. Otherwise it's merged to the `develop` branch as usual.

Hotfixes made on `master` can be applied on `develop` directly, via cherry-pick.

## Releases

All releases are made from `master` only. No release branches, typically used on Gitflow, are used here.

The main reason for it is to allow developers to integrate their changes to `develop` as "block-free" as possible, but still keeping `master` in a stable state.

Additionally, `master` and `develop` have totally distinct histories, where the exchange of code happens only via cherry-picking. This differs from Gitlab where both branches share a common history via merge requests.

A release should be done via a merge requet on master. The branch name for the merge request should be in the form: `feature/release-x.x.x`.

Every release, final or candidate (RC), should be tagged accordingly. Gitlab will do the tagging automatically during the release process.

The tags have for format: `release/x.x.x`, where `x.x.x` is the version number.

## Versioning

We use [semantic versioning](http://semver.org). Release candidates have a `-RCX` appended to the version number.

## Backporting

We support only the most recent released version. Therefore no backporting of features or bugfixes are done on older versions.
