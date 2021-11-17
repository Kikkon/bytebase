# General

1. Put more thought on data modeling and naming.
1. Write comment and use English.
1. If the change is beyong trivial, write informative PR title and description.


# How to build a feature

Split into smaller changes. This makes reviewer life easy and you will get faster feedback.

If you are working on an end-to-end feature including both backend and frontend, the usual steps to follow are:

1. If it requires schema change, design the schema first (you may need to discuss with peers).
1. Design the API. [Our API style guide](https://github.com/bytebase/bytebase/blob/main/docs/api-style-guide.md)
1. Finish the rest.

Ideally, you will split the schema change, API change and the rest into separate PRs. If you put them together, then if the schema requires a change after the review, it will end up with a lot of code changes. For obvious schema changes, you can still choose to put them in a single PR.

All in all, figure out the schema/data model first before moving forward and use your judgement to decide whether to split the change into separate PRs.

## Example commits

1. An end-to-end example showing you the code touched when adding a field to the schema and populate it all the way to UI: [Add path field to backup setting](https://github.com/bytebase/bytebase/commit/a7c28a4fefb2c2cff0c1ed9bb7fc043a47f535cd#diff-e547f2c710d4d67f2887ee13f4361d35537404829114e9c10d6aa5f48b3179dc)