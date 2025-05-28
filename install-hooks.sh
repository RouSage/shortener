#!/bin/sh

cp ./hooks/pre-commit.sh .git/hooks/pre-commit
cp ./hooks/pre-push.sh .git/hooks/pre-push

chmod 755 .git/hooks/pre-commit
chmod 755 .git/hooks/pre-push
