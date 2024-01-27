# firefox-tabs

This program stores and loads the tabs of Firefox instances.

- [Current status](#current-status)
- [Why does this program exist?](#why-does-this-program-exist?)
- [Installing](#installing)
- [Usage](#usage)
  - [store](#store)
  - [load](#load)

## Current status

It works on my machines. I may make this prettier.

## Why does this program exist?

Firefox Sync is unreliable: tabs that are open in another browser sometimes
don't appear. When it doesn't work, there's nothing I can about it through
Firefox Sync itself. I've grown tired of the situation.

This trivial program stores and loads Firefox tabs reliably. It serves my needs.

## Installing

This requires a Go toolchain.

```sh
go install github.com/adamroyjones/firefox-tabs@latest
```

## Usage

The program has two commands: store and load.

This program doesn't cover synchronisation or automation. For those ends, I'd
use [syncthing](https://syncthing.net) and a systemd unit.

### store

```sh
firefox-tabs store
```

will, for each profile for the current machine's instance of Firefox, write out
the open tabs to

```
~/.config/firefox-tabs/data/<hostname>/<profile>.json
```

by parsing the file used to restore the profile.

### load

```sh
firefox-tabs load
```

will read out all of the files from

```
~/.config/firefox-tabs/data/
```

and open them up as a table in Firefox.
