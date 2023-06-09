import os
import praw
from typing import Any, Dict, Optional, Tuple
import requests
from datetime import date, datetime
import logging
import json
from time import sleep
import re

headers = {
    'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36'
}

PATH_FILE = f'comments - {datetime.now()}.txt'

logging.basicConfig(
    filename=PATH_FILE,
    force=True,
    format='%(asctime)s %(levelname)s %(message)s',
    level=logging.INFO,
)
logger = logging.getLogger('comments')


def get_season(dt: datetime) -> Tuple[str, str]:
    Y = 2000
    seasons = [('winter', (date(Y,  1,  1),  date(Y,  3, 20))),
            ('spring', (date(Y,  3, 21),  date(Y,  6, 20))),
            ('summer', (date(Y,  6, 21),  date(Y,  9, 22))),
            ('autumn', (date(Y,  9, 23),  date(Y, 12, 20))),
            ('winter', (date(Y, 12, 21),  date(Y, 12, 31)))]

    if isinstance(dt, datetime):
        dt = dt.date()
    num_year = dt.year
    dt = dt.replace(year=Y)
    season = next(season for season, (start, end) in seasons
                if start <= dt <= end)
    if season == 'winter':
        if dt <= date(Y, 3, 20):
            year = f"{(num_year - 1) % 2000}_{num_year % 2000}"
        else:
            year = f"{(num_year) % 2000}_{(num_year + 1) % 2000}"
    else:
        year = str(num_year)

    return year, season


def get_file_path(submission) -> Optional[str]:
    match = re.match(
        "(.+)(| - )Episode (\\d+)(.+)discussion",
        submission['title'],
        re.IGNORECASE,
    )

    if match is None:
        logger.info(f"Did not match {submission['title']} skipping link to post: {submission['url']}")
        return None

    year, season = get_season(datetime.fromtimestamp(submission['created_utc']))
    path = f'threads/{season}_{year}/'
    if not os.path.isdir(path):
        logger.info(f"Creating folder {path}")
        os.mkdir(path)

    filename = match.group(1) + match.group(3) + ".json"
    filename = filename.replace("/", "-")
    full_path = path + filename
    if os.path.exists(full_path):
        return None
    return full_path


def _comment_forest_to_dict(comment_forest):
    all_comments = []
    for comment in comment_forest:
        new_comment = {
            key: value
            for (key, value) in comment.__dict__.items()
            if key[0] != '_'
        }

        new_comment['replies'] = _comment_forest_to_dict(comment.__dict__['_replies'])
        all_comments.append(new_comment)

    return all_comments


def comment_forest_to_dict(comment_forest):
    return {'data': _comment_forest_to_dict(comment_forest)}


if __name__ == '__main__':
    # file generated by pushshift_get_submission.py
    with open("all_submissions_2019_2023.json", "r") as f:
        submissions = json.loads(f.read())

    with open("lurker-bot.json", "r") as f:
        bot_info = json.loads(f.read())

    reddit = praw.Reddit(
        user_agent=bot_info["user_agent"],
        client_id=bot_info['client_id'],
        client_secret=bot_info['client_secret'],
        username=bot_info['username'],
        password=bot_info['password'],
    )

    for i, submission in enumerate(submissions['data']):
        logger.info(
            f"({i + 1}/{len(submissions['data'])}) "
            f"Currently at {str(datetime.fromtimestamp(submission['created_utc']))} "
        )
        logger.info(f"URL: {submission['url']}")

        if (file_path := get_file_path(submission)) is not None:
            logger.info("Requesting thread")
            thread = reddit.submission(submission['id'])

            logger.info("Requestions all comments")
            thread.comments.replace_more(limit=None)

            logger.info("Writing to file")
            with open(file_path, "w") as f:
                json.dump(
                    comment_forest_to_dict(thread.comments),
                    f,
                    default=str,
                )
        else:
            logger.info(f"File of title {submission['title']} already exists")
