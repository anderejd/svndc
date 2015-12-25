# svndc
Subversion Diff Commit - WORK IN PROGRESS

This program automates SVN for a specific purpose: 
Commit a local unversioned directory to SVN with proper history using a one-liner with robust error handling. (eg. Jenkins CI build output)

The following steps are automated:
  1. Checkout a temporary working copy. (or update an existing working copy)
  2. Delete all except .svn in working copy.
  3. Copy all files/directories from local unversioned directory to working copy.
  4. Add all new files/directories.
  5. Delete missing files/directories in working copy. (this is the main reason this tool exists)
  6. Commit.
  7. Delete working copy. (optional)

All global SVN options are passed through to the svn subprocess.

