# CraftDocs Workflow

Search in Craft document with Craft Search Index.

---

#### Alternative references

[Craft extension](https://www.raycast.com/bgnfu7re/craftdocs) for [Raycast](https://www.raycast.com/).

---

## Install
Download
[the latest release](https://github.com/kudrykv/alfred-craftdocs-searchindex/releases/tag/v0.1.2),
double-click it and proceed with Alfred instructions.
Use `amd64` for Intel chips and `arm64` for M1 chips.

## Search
Run `cs <query>` to look for documents.
It opens the page with the result.

![](search_1.png)

![](search_2.png)

## Authorization
The first time you use the workflow after install or update, you will see the security warning:
<img alt="unidentified developer security warning" src="security_warning.jpeg" width="400">

This is a quirk of MacOS 10.15 and above.

After seeing this warning, you have to go to
`System Preferences > Security & Privacy > General`
and click the new button that has appeared to allow the executable to run.
You then have to run it again, and you will see this security warning again,
but now it will have a new button that lets you allow the executable to run.
