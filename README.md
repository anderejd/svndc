# svndc
Subversion Diff Commit

This program automates SVN for a specific purpose: 
Commit a local unversioned directory to SVN with proper history using a one-liner with robust error handling. (eg. Jenkins CI build output storage)

The following steps are automated:
  1. Checkout a temporary working copy. (or update an existing working copy)
  2. Delete all except .svn in working copy.
  3. Copy all files/directories from local unversioned directory to working copy.
  4. Add all new files/directories.
  5. Delete missing files/directories in working copy.
  6. Commit.
  7. Delete working copy. (optional)

All global SVN options are passed through to the svn subprocess.

```
usage:
svndc --src-path PATH --repos-path URL --wc-path PATH --message "There are only 12 cylon models." --username GBaltar --password 123Caprica ...

--help           Print syntax help
--src-path       Path to directory with files to commit
--repos-url      Target SVN repository URL (commit destination)
--wc-path        Working copy path. This path will be created by svn
                 checkout, if it does not exist. Files from --src-path 
                 will be copied here. Files not present in --src-path
                 will be svn-deleted in --wc-path.
--wc-delete      Will delete --wc-path after svn commit.
--message        Message for svn commit.
--self-test      Requires svnadmin. Will create a local repository in 
                 the directory ./self_test/repos and use for tests. The
                 directory ./self will be deleted when tests complete.

SVN Global args (see svn documentaion):

--config-dir ARG
--config-options ARG
--no-auth-cache
--non-ineractive
--password ARG
--trust-server-cert-failures ARG
--username ARG
```
