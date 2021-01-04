# 2. Branching Model

Date: 2020-11-24

## Status

Accepted

## Context

Prior to the 1.0 release, our branching model consisted on:

- `master`, as our development branch. Any new merge request would be done from it.
It's not stable and it's sometimes accidentally broken.

- `feature/*` and `fix/*` branches, used while a new feature, complete or not, is being developed.
Once approved, it's merged to master.

- `feature/release-*`, based on master, where any fixes for a release candidate are done. A release is made from this branch as well.

During the execution of the 1.0 release, having a separated release branch for it demonstraded to be quite challenging,
as we made a few last-minute changes to it during the first RC and the final release, causing some confusion on the team
regarding the process.

This in addition with the branch model caused issues including:

- should a change be made on master or on the release branch if it affects both development version and the to-be-release one?
- was some specific change included in the release? Was it supposed to be in the release? Given that a bug is found,
does it affect a release, being a blocker to it?
- a developer can be easily blocked if other developers who review their code are not available (on sick leave or on holidays, for instance),
as any merge to master needs to be reviewed.
- should a developer squash commits when merging them to master? Squashing making reverting changes easier, but generates bigger commits,
making finding where bugs/issues have been introduced harder to track.
- new strings that are added are made translatable only when a release is made, making translations unavailable by definition for them when a new release is done. This happens because translations are performed on `master` for code that has already been landed.
As changes on `master` happen quite often, it's difficult to enforce a "string freeze" without blocking developers of landing new changes there.

## Decision

On a discussion including team members with different perspectives, consisting on project and product management, marketing and engineering,
we decided to change the branching model somehow inspired on the popular [Gitflow](https://www.atlassian.com/git/tutorials/comparing-workflows/gitflow-workflow) where, briefly:

- there's a `develop` branch that consists any approved merge request for new features. Although review is needed, a developer can merge to it
if no reviewers are available.
- there's a `master` branch, where the releases are made from. It must go through a higher quality assurance process, especially if it hasn't been peer reviewed. A feature is said to be "Done" only after merged to it.
- there are feature branches, based either on `develop` or `master`.
- all release candidates and final releases are made on `master`, eliminating the need for release branches.
- `master` is long-lived and stable. It contains only merge requests from develop or hotfixes needed during the releasing process
(for bugs found in the RCs or for last minute changes addressing marketing issues).
- a string freeze will be made where any new strings are forbidden to land on `master`, giving time for the translation folks to work on them before the release.

More information about is available in a [further ADR document](0003-translations-workflow.md) and in the `TRANSLATION.md` file.

The main differences from Gitflow are:

- the lack of the release branches (especially during). In the current proposal, all releases are made from master.
- `develop` and `master` have totally different histories. On Gitflow things from `develop` are periodically merged to `master`,
whereas in the current proposal, code is "copied" to `master` via cherry-picking.

For detailed information, please consult the file `docs/DEVELOPMENT_FLOW.md`.

## Consequences

The revieweing process will now be a two-step process, where for a feature to land on `master`, it'll need to go from a feature branch to `develop`
and then from `develop` to `master`.

This brings some extra work for the team, but considering that a feature proposed to be landed on `master`
has already gone through intense reviewing process on develop, the review to land on `master` will potentially
require much less time and intelectual effort.

On the other hand it can also be quite error prone, in case commits from multiple features are interleaved to `develop`.

This happens because landing a feature from `develop` to `master` requires cherry-picking only the commits from such feature.
In case there are many commits, this process can be a bit tedious and error prone.

The extra complexity has the benefit of allowing developers to continue working on new features without being blocked when other potential reviewers are not available, making occasional time-offs and leave/holidays planning a less stressful process :-)

This approach allows us to always know where to find a release-blocking issue, and where to fix it. (spoiler: it's on master)

The commit history of `master` is more likely to "look good", meaning it's linear with fewer branches. Reverting commits on it is also easier.

On the other hand, such approach makes it more difficult to maintain older releases with fixes, as there's only one `releasable`.
But, considering that we have no plans to support versions other than the current one, this is not an issue at the moment.

Additionally it increases the probability of missing commits on cherry-picking. Luckily git is very likely to point out errors if it happens,
and the result code is like to not build/test properly in case this happens.

To prevent entire features of be "forgotten" to be added to `master`, we enforce a policy where a feature is considered `Done` only when it's landed on `master`.

The proposed workflow is also very unusual for new developers accostumed to Gitflow or other polular flows, possibly creating some initial barrier and/or resistence from such developers.

Also, as such approach feels quite novel and non battle tested, there might be issues yet to be discovered, which can create tension in the team.

I am not aware if such approach has been successfully used or even if it has a name.
