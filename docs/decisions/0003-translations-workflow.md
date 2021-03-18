# 2. Translations Workflow

Date: 2020-11-24

## Status

Accepted

## Context

Since version 0.0.6 Lightmeter supports i18n, and translations in various natural languages. 
Translations for the application and components are managed using Weblate, accessible at translate.lightmeter.io.

For contribution guidelines, please consult the file `TRANSLATIONS.md`.

New strings that are added are made translatable via Weblate when a release is made. New translations are commited by Weblate to the Control Center repository on Gitlab via Git, directly to the master branch. 

This approach, when combined with the existing branching model, documented in ADR 0002, introduces the following problems:

1. New strings are not available for translation before a release is done, and strings related to new features are not translated in the first release which includes them. 
2. Direct commits from Weblate bear a risk of breaking master. 

## Decision

On a discussion including team members with different perspectives, consisting on project and product management, marketing and engineering, we decided: 

1. Change the Weblate commit workflow: Weblate should not commit changes directly to master but instead it should create a MR. 
2. A "String Freeze" should be enforced during the "Feature Freeze" phase, with tolerance on cases of typos and/or clarifications or improvements of existing text. 

## Consequences

Weblate creating a MR for new translations, instead of commiting to master, makes it technically possible to enforce a "String Freeze" on master, creating an opportunity for Lightmeter contributors to add translations for the new features during the RC phase. This means that in stable releases, new features can be used in different supported languages on the same day those features are first released. 

