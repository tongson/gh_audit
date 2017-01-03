Github organization member/user auditing tool
=============================================

Generates a CSV for audits.

    $ go get github.com/tongson/gh_audit/...
    $ GITHUB_TOKEN="de4dbeefbad1d34" GITHUB_ORG="Configi" gh_audit audit.csv
    $ cat audit.csv
    ID,Login,Name,Type,Teams
    4187,tongson,User,Owners
