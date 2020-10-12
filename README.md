subscribe-bot
=============

Subscribes to OSU map updates and stores versions for later review.

Please don't run a separate bot, the official one is `subscribe-bot#8789`. If
you want to contribute or test the bot, then here are instructions on how to
run it:

How to run
----------

1. Build the bot using a Go compiler that supports modules (1.11 or higher).
   Running `go build` in the root of the repo should work.
1. Create a configuration file called `config.toml` (can be called something
   else as long as you pass it into the executable as a command-line argument).
    - `client_id` (int) and `client_secret` (string) are oauth-related settings
    you can obtain from the OSU settings page.
    - `bot_token` (string) is Discord's bot auth{entication,orization} token,
    you can get that from Discord developers' page.
    - `repos` (path) is a path to where map repositories should be stored.
1. Run the executable, passing `-config {path}` in case you want to use a
   different config file than `config.toml`.

License
-------

[GPL3][1]

[1]: https://www.gnu.org/licenses/gpl-3.0.en.html
