# Translating Lightmeter

For managing the translation process we use [Weblate](http://translate.lightmeter.io). All translation changes are handled via this system.

Merge requests including direct edits to translation files (`.po` and `.pot`) do not fit this workflow, and so cannot be accepted. However you can [download](https://docs.weblate.org/en/latest/user/files.html) those files from Weblate, edit them, then [upload them](https://docs.weblate.org/en/latest/user/files.html), if you prefer using your own editing tools.

## Getting help

Ask questions in the [Translators section](https://discuss.lightmeter.io/c/translation/9) of the Lightmeter discussion forum.

## Using Weblate

The first step is to get familiar with Weblate.

### Sign In

To contribute translations at <http://translate.lightmeter.io> you must create a Weblate account. You may create a new account or use any of the supported sign in services.

### Language Selection

Lightmeter is being translated into several languages.

1. Start by selecting the part of Lightmeter which you want to translate
1. Then select the language you want to translate into form the list
1. If your desired language is not listed, then click "Start new translation" to create a new one
1. Click "Translate" to start translating available words and sentences

### Applying translations

Translations which are stored in Weblate are automatically added to Lightmeter software repositories. Weblate commits changes to translation files via Git, and then those files are used by Lightmeter software components. Updated translations are included in each Lightmeter release -- translations are not updated between Lightmeter updates (you must wait until the next Lightmeter release to use the most recent translations with applications). 

If you want to test the most recent translations between releases, then you can download a development version from GitLab and compile it.

## General Translation Guidelines

Be sure to check the following guidelines before you translate any strings.

### Formality

The level of formality used in software varies by language:

| Language | Formality | Example |
| -------- | --------- | ------- |
| French | formal | `vous` for `you` |
| German | informal | `du` for `you` |

You can refer to other translated strings and notes in the glossary to assist
determining a suitable level of formality.

### Inclusive language

We ask you to avoid translations which exclude people based on their gender or
ethnicity.

In languages which distinguish between a male and female form, use both or
choose a neutral formulation.

For example in German, the word "user" can be translated into "Benutzer" (male) or "Benutzerin" (female).
Therefore "create a new user" would translate into "Benutzer(in) anlegen".
