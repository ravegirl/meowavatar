"""
Integration tests for meowavatar.
"""

import os
import requests

BASE_URL = os.environ.get("BASE_URL", "http://app:8080").rstrip("/")


def get(social: str, identifier: str) -> requests.Response:
    url = f"{BASE_URL}/{social}/{identifier}"
    resp = requests.get(url, timeout=15)
    return resp


def assert_image(resp: requests.Response) -> None:
    assert (
        resp.status_code == 200
    ), f"Expected 200, got {resp.status_code}: {resp.text[:200]}"
    ct = resp.headers.get("Content-Type", "")
    assert ct.startswith("image/"), f"Expected image Content-Type, got: {ct!r}"
    assert len(resp.content) > 0, "Response body is empty"


def test_github_torvalds():
    assert_image(get("github", "torvalds"))


def test_github_case_insensitive():
    r1 = get("github", "torvalds")
    r2 = get("github", "Torvalds")
    assert r1.status_code == 200
    assert r2.status_code == 200


def test_github_not_found():
    resp = get("github", "this-user-does-not-exist-xyzzy-99999")
    assert resp.status_code in (404, 502)


def test_reddit_spez():
    assert_image(get("reddit", "spez"))


def test_steam_vanity_gaben():
    assert_image(get("steam", "gaben"))


def test_steam_not_found():
    resp = get("steam", "this-vanity-does-not-exist-xyzzy99999")
    assert resp.status_code in (404, 502)


def test_twitch_xqc():
    assert_image(get("twitch", "xqc"))


def test_twitch_not_found():
    resp = get("twitch", "thischannel_doesnotexist_xyzzy99999")
    assert resp.status_code in (404, 502)


def test_twitter_jack():
    assert_image(get("twitter", "jack"))


def test_x_alias():
    assert_image(get("x", "jack"))


def test_twitter_not_found():
    resp = get("twitter", "thisdoesnotexist_xyzzy99999abc")
    assert resp.status_code in (404, 502)


def test_telegram_paul():
    assert_image(get("telegram", "paul"))


def test_telegram_not_found():
    resp = get("telegram", "thisdoesnotexist_xyzzy99999abc")
    assert resp.status_code in (404, 502)


def test_unsupported_social():
    resp = get("myspace", "tom")
    assert resp.status_code == 404


def test_cache_hit():
    r1 = get("github", "torvalds")
    assert r1.status_code == 200
    r2 = get("github", "torvalds")
    assert r2.status_code == 200
    assert r2.headers.get("X-Cache") == "HIT"


def test_ratelimit_headers():
    resp = get("github", "torvalds")
    assert "X-RateLimit-Limit" in resp.headers
    assert "X-RateLimit-Remaining" in resp.headers
