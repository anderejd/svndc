# svndc
Subversion Diff Commit - WORK IN PROGRESS

This program will automate svn for a specific purpose, to let Jenkins CI commit build output to svn with proper history. (instead of remote delete + import)
The following steps will be automated:
1. Checkout a temporary working copy.
2. Add changed and new files/dirs, delete missing files/dirs.
3. Commit.
4. Delete temporary working copy.

The program will send all command line arguments through to the svn subprocess.

