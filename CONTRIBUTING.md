# Contributing

Contributions of all kinds are welcome for all Suborbital projects. There are two rules that must be adhered to when making contributions:

1. All interaction with Suborbital on GitHub or in other public online spaces such as Discord, Slack, or Twitter, must follow the [Contributor Covenant Code of Conduct](https://github.com/suborbital/meta/blob/master/CODE_OF_CONDUCT.md) which is kept up to date in the `suborbital/meta` repository, and linked to here.

2. Any code contributions must be preceded by a discussion that takes place in a GitHub issue for the associated repo(s). Please do not submit Pull Requests before first creating an issue and discussing it with the Suborbital team or using an existing issue. This includes all changes to the contents of Suborbital Git repositories, except for the following content: documentation, README errors, additional example code, additional automated tests, and additional clarifying information such as comments. The Suborbital team can choose to close any Pull Request that does not have an appropriate issue.

Beyond all else please be kind, and welcome to the Suborbital family of projects! We're really glad you're here.

## HACKTOBERFEST

Hacktoberfest contributions are welcome! For this project, we want to showcase some interesting uses for Sat, so we would love to see some new examples added to the repo! Examples are an extremely effective way to help developers learn how to take advantage of WebAssembly on the server.

To make a Hacktoberfest contribution, fork and clone this repo and use our `Subo` CLI tool ([here](https://github.com/suborbital/subo)) to create a new example:

```bash
# you can use --lang typescript, rust, or swift!
# pick a name that represents what the function does
subo create runnable my-new-function --dir ./examples --lang typescript
```

Next, build Sat:

```bash
make sat/dynamic
```

Then write a function that does something cool! You can find documentation for writing functions [here](https://atmo.suborbital.dev/runnable-api/introduction). Ideas could include sending a message to a Discord server, fetching data from an API you're familiar with, or anything else you can think of!

Test your new function by building it (make sure you have Docker installed!):

```bash
subo build ./examples/my-new-function
```

And then run it:

```bash
SAT_HTTP_PORT=8080 .bin/sat ./examples/my-new-function/my-new-function.wasm
```

Make a POST request to `localhost:8080` and your function will run!

Finally, open a PR with a title like this:
> `[HACK] New example: {your function's name}`

And we'll get it merged!
