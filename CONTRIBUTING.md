# Contributing

First fork the repo to your account.

Then setup your local environment:

```shell
# Subsitute with your github user
user="your github username"

# Clone it
git clone git@github.com:$user/virsnap.git
# or: git clone https://github.com/$user/virsnap.git

# Set remote
git remote add upstream git@github.com/joroec/virsnap.git
# or: git remote add upstream https://github.com/joroeck/virsnap.git

# Don't push to upstream
git remote set-url --push upstream no_push

# Check it all makes sense
git remote -v
```

Update your local master:

```shell
cd virsnap
git fetch upstream
git checkout master
git rebase upstream/master
```

Create a feature branch:

```shell
git checkout -b feature_branch
```

Keeping your local branch in sync:

```shell
# While on your feature_branch
git fetch upstream
git rebase upstream/master
```

After adding your changes, commit them:

```shell
git commit
```

Then push your changes to your local repository. If you rebased during
development you might need to force-push:

```shell
git push -f
```

Lastly go to your vork on Github and create a pull request. Assign one of the
authors from the `AUTHORS` file to review it.