# Building

Very simple.

    make clean
    make issuebot

# Running


## Slack setup
First, one must have a Slack workspace that one can install applications in.
Within that workspace one must (currently) create a slackbot for the
workspace and create a slack command for it.

First visit https://api.slack.com/slack-apps to get to the root of the
documentation.  From that same page you can click on the "Create a Slack
App" buttons to start create the application.

Once the application is set up in your workspace you go to
https://api.slack.com/apps and click on the name of your application.
(I call mine 'issuebot'.)  From there you will see an "Add Features and
Functionality" section which if you click on it will have a section for
"Slach commands".  Alternately if clicking on the name of your
application takes you to https://api.slack.com/apps/ABCDEFGH then going
to https://api.slack.com/apps/ABCDEFGH/slash-commands will bring you to
the same dialog.

In that dialog you need to create a command named "/issue".  You can put
in a dummy URL field for now, but keep this page (which I'll refer to as
the "Slash Command Page" later saved for future reference.  You should
now be ready to go on to the next part.

TODO:  verification tokens

TODO:  Oauth2 support

## Github Setup
The issuebot assumes that you are dealing with a Github repository that
tracks issues through Github.  In order for the issuebot to access the
repository beyond queries it needs credentials.  Either you as owner of
the repository or a separate account that is a member of the repository
will do.  Either way, logged in as that user to go:
https://github.com/settings/tokens/new to generate a fresh token for
this account.  You need to give the token "repo" level scope access.
(Click the "repo" checkbox when creating.)  Remember to copy the token
hext string immediately after creation otherwise you will not be able
find it again later and will have to generate a new one.  This token
will be what you use in the ISSUEBOT\_AUTH environment variable for the
issuebot below.

TODO:  Github Oauth tokens


## Ngrok
Issuebot can run on a public server, but for now while in development
I'm assuming that it will run in a private container on a private
machine with 'ngrok' providing the public facing URL.  The examples
below also currently assume that you do not have a paid ngrok account
and so get a dynamic URL every time you start ngrok.   If this is not
the case, then you can skip the last few steps for extracting the ngrok
URL and putting it in Slack slash command configuration.  (Instead just
put in your static ngrok URL.)

However, if one is operating in this manner one must also change the
`docker run ... ngrok` invocation below to add your `ngrok` credentials
as environment variables.  See https://hub.docker.com/r/wernight/ngrok/
for details about how to do this.


## Running the Program (in Docker)
These steps presume that you have Docker installed on your platform.
First, build a fresh docker image with the issuebot binary.  Rename
the tag as desired.

    $ docker build -t ctelfer/issuebot .

Create a docker network for communicating between 'ngrok' and issuebot.

    $ docker network create --driver bridge issuebotnet

Start the issuebot in its container.
  * The invocation below assumes that you have created github
    account "user" for the issuebot and that the repository
    it will manage is owned by "owner" and named "repo".
  * In practice "user" and "owner" could be the same if you
    are just giving the issuebot an authentication token from
    the owner.
  * "token" is the authentication token mentioned above under
    "Github Setup"

    $ docker run -dt --rm --name=issuebot --network=issuebotnet \
            --env "ISSUEBOT_USER=user" \
            --env "ISSUEBOT_REPO=owner/repo" \
            --env "ISSUEBOT_AUTH=token"\
            ctelfer/issuebot

Run ngrok and connect it to the issuebot network.  You will need to add
credentials here as environment variables if you are running `ngrok`
with a paid account.  This and the following steps assume you are just
using the free service for now.

    $ docker run --rm -itd -p 4040 --name=ngrok-issuebot --network=issuebotnet \
            --env="NGROK_PORT=issuebot:80" wernight/ngrok

Obtain the public ngrok URL that one uses to connect to the issuebot.

    $ NGROK_MGMT_PORT=$(docker port ngrok-issuebot 4040 | sed -e 's/^.*://')
    $ curl -s http://localhost:$NGROK_MGMT_PORT/status | 
            grep ngrok.io |
            sed -e 's/^.*URL\\":\\"//' -e 's/ngrok\.io.*/ngrok.io/' -e 's/http/https/'

Finally, go to https://api.slack.com/apps/ABCDEFGH/slash-commands (see
above) and edit your "/issue" command to point to "URL/issue" where
"URL" is the value returned by the `curl` command above.  For example,
if the above command returned `https://d12345678.ngrok.io` then one
should put `https://d12345678.ngrok.io/issue` in the URL for the slash
command field and pressing the 'Save' button.

At this point you should be able to test that everything works by going
into your slack workspace and typing `/issue help` which should return a
list of `/issue` subcommands.
