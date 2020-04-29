# HOW TO RELEASE

We use semantic versioning (see http://semver.org).

- Change the file VERSION.txt with the new version number.
- Create a plain text file ../release_notes/{VERSION}, where {VERSION} is the
same version number chosen in the previous step.

Commit and push your changes.

After Gitlab CI for your commit finishes, click on the button ▶ and "do-release" for the correspondent pipeline.

After that, you'll have a new release in the Releases page.

Simple like that :-)
