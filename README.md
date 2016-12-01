ghcli
=====
`ghcli` is a project that contains CLIs and libraries that perform operations on GitHub projects and repositories.

ghlicense
---------
`ghlicense` is a tool that writes, verifies and applies standard licenses for GitHub projects. It obtains the license
templates from GitHub.com using the GitHub.com license API.

The `print` and `write` commands can be used to view or generate the correct canonical license files for a given
license (with the author and copyright year filled out properly when they exist).

The `verify` and `fix` commands can be used to ensure that existing repositories use the correct version of a license.
These commands assume that the repositories they operate on already have license files that are either correct or mostly
correct (at least, correct enough for GitHub to infer the type of the license based on the content). These commands act
as clean-up tasks that verify that verify that the formatting and content of the license files are consistent and
correct.

Note that, due to the manner in which the GitHub license APIs work, the `verify` and `fix` commands do not work on forks
(forked repositories do not return license information). These commands should be run directly on the source
repositories instead (the `fix` command will open PRs in the forked repository to perform the fix).

### Installation
```
go get github.com/nmiyake/ghcli/ghlicense
```

### Usage
The `--help` flag can be used to print a list of the available commands and flags. If the `--github-token` flag is used,
the provided token is used as the OAuth token for all of the API calls, otherwise, calls are made anonymously. The `fix`
command requires a token because it needs an authenticated user as which to open PRs.

### Rate Limit
Print the API rate limit (either for the provided token or for the current anonymous host):

```
> ghlicense rate-limit --github-token {token}
Remaining requests: 4995/5000
Rate limit resets:  09:52:37 PST Sun Nov 27 2016
```

### List
Print IDs and aliases of all licenses:

```
> ghlicense list
ID              ALIASES
AGPL-3.0        agpl
Apache-2.0      apache
BSD-2-Clause    bsd-2
BSD-3-Clause    bsd-3
EPL-1.0         epl
GPL-2.0
GPL-3.0         gpl
LGPL-2.1
```

#### Print

Print the content of a license:

```
> ghlicense print apache
                                 Apache License
                           Version 2.0, January 2004
...
```

#### Write

Write the contents of of a license to a file (default is LICENSE):

```
> ghlicense write mit --author="Nick Miyake"
```

#### Verify

Verify that the license file in a repository is correct:

```
> ghlicense verify --github-token {token} --user nmiyake --author="Nick Miyake"
Verifying license for repository bar (1/30, page 1/3)...incorrect
Verifying license for repository foo (2/30, page 1/3)...OK
...
4 repositories had correct license files:
        bar
        ...
5 repositories had incorrect license files:
        foo: actual content of license does not match expected content:
                --- Expected
                +++ Actual
                @@ -202,0 +203,2 @@
                +Modified
                +
        ...
```

#### Fix

Open PRs to fix licenses in a repository:

```
> ghlicense fix --github-token {token} --user nmiyake --author="Nick Miyake" foo
Verifying license for repository foo (1/1)...incorrect
Open PR for fix (y/n): y
User has push permissions to repository
Creating tree...OK
Creating commit...OK
Creating branch...OK
Creating pull request...OK
Examined 1 repository and opened 1 pull request.
```

ghspec
------
`ghspec` is a tool that enforces GitHub repositories to follow a declarative specification. Repositories are specified
using YML or JSON and the definition contains information such as the repository description, owners and license type.
`ghspec` can then be used to verify that the repositories belonging to a user or organization follow the defined
specification. The tool can also be used to open PRs to fix any repositories that do not conform to the specification.
The tool can also generate a specification file for existing repositories to use as a baseline.

### Installation
```
go get github.com/nmiyake/ghcli/ghspec
```

### Usage
The `--help` flag can be used to print a list of the available commands and flags. If the `--github-token` flag is used,
the provided token is used as the OAuth token for all of the API calls, otherwise, calls are made anonymously. The `fix`
command requires a token because it needs an authenticated user as which to open PRs.

### Rate Limit
Print the API rate limit (either for the provided token or for the current anonymous host):

```
> ghspec rate-limit --github-token {token}
Remaining requests: 4995/5000
Rate limit resets:  09:52:37 PST Sun Nov 27 2016
```

### Create
Create a specification file for existing repositories owned by a user or organization.

```
> ghspec create --user nmiyake --github-token {token} repositories.yml
Generating definition for nmiyake/bar...done
Generating definition for nmiyake/foo...done
...
Wrote definitions for 3 repositories to repositories.yml
```

### Verify
Verifies that the repositories owned by a user or organization conform to the provided specification.

```
> ghspec verify --user nmiyake --github-token {token} repositories.yml
Verifying repository bar against definition (1/30, page 1/3)...OK
Verifying repository foo against definition (2/30, page 1/3)...OK
...
60 repositories OK
        nmiyake/bar
        ...
1 repository differed from definition:
        nmiyake/foo:
                LICENSE content (Apache License 2.0):
                        --- Expected
                        +++ Actual
                        @@ -189 +189 @@
                        -   Copyright {yyyy} {name of copyright owner}
                        +   Copyright 2016 Nick Miyake

```

### Apply
Applies the provided specification to the repositories owned by a user or organization. Opens pull requests or makes API
calls as necessary to ensure that the repositories match the provided specifications.

License
-------
This repository is made available under the [MIT License](https://opensource.org/licenses/MIT).
