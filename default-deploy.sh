#!/bin/sh

DIR=/data/$APP-$BRANCH

echo Abouty to deploy. APP=$APP, BRANCH=$BRANCH, DIR=$DIR

if [ ! -d "$DIR" ]; then 
  git clone -b $BRANCH --single-branch https://github.com/Alars-ALIT/busybox-go-webapp.git $DIR
fi

cd $DIR
git pull