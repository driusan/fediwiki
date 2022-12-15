# FediWiki

A (WIP) ActivityStreams based wiki.

You can currently log in via OAuth on a Mastodon and any logged
in user can edit any page.

Done:
- [x] Login with OAuth
- [x] Create new pages when logged in
- [x] Update pages when logged in

TODO:
- [ ] Create page actors that can be followed
- [ ] Send out a note with the diff when a page edit is proposed (to owner) or accepted (to followers) 
- [ ] Record history of edits somewhere
- [ ] Require approval of edits from page owner or admin
- [ ] Federate the Article content to followers. Send an Update activity for the article to followers when a page change is accepted.
- [ ] Implement page owner and admin
- [ ] Improve CSS / styling
- [ ] Create "Talk" page for diffs where they can be discussed
- [ ] Create "Talk" page for article for any mentions of it.
- [ ] Remove Mastodon assumptions where possible (ie use .well-known/openid-configuration instead of hardcoded endpoints if it exists)
- [ ] Comment the code, follow best practices, etc

