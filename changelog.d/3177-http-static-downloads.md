### Fixed
- **Large audiobook downloads no longer stop at two minutes** (#3177). A book or audiobook download was cut off when it had been running for two minutes, however steadily the client was reading, which hit big files on a slow link. Downloads now run as long as the client needs, and resuming with a range request still works. The OPDS download and the Calibre bridge file transfer are covered too.

### Changed
- **The web app's files are cached for a year** (#3177). Build files under `/assets/` carry a content hash in their name, so the browser and any proxy in front of Bindery can keep them without asking again. The app shell, favicon and manifest are still checked every time, so an upgrade reaches you straight away.
- **Audiobook zips build much faster** (#3177). Audio, ebook and image files are stored in the zip as they are instead of being compressed a second time. A zip of an 800 MiB audiobook folder went from about 14 seconds of CPU to well under a second. Text, cue sheets and PDF are still compressed.
- **Metadata lookups keep more connections open** (#3177). Open Library, Hardcover, Google Books and the other metadata providers keep up to 16 idle connections per host instead of 2, so a burst of parallel lookups reuses them instead of reconnecting.
