import requests
from datetime import datetime
import logging
import json
from time import sleep

headers = {
    'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36'
}

PATH_FILE = f'submissions - {datetime.now()}.txt'

logging.basicConfig(
    filename=PATH_FILE,
    force=True,
    format='%(asctime)s %(levelname)s %(message)s',
    level=logging.INFO,
)
logger = logging.getLogger('submissions')

url = "http://api.pushshift.io/reddit/search/submission?subreddit=anime&size=500&title=episode,discussion&before="
before = "1542981747"
all_submissions = json.loads('{"data":[]}')
total_it = 100

try:
    for i in range(total_it):
        logger.info(f"({i}/{total_it}) Total size: {len(all_submissions['data'])}")
        logger.info(f"Last update from {str(datetime.fromtimestamp(int(before)))} - {before}")

        r = requests.get(url + before, headers=headers)
        if r.status_code == 200:
            all_submissions['data'].extend(r.json()["data"])
            before = str(r.json()["data"][-1]["created_utc"])
        else:
            logger.warning(f"Response code not 200 - {r.status_code} instead, trying again")
        sleep(2)
except Exception as e:
    logger.error(f"Error while in execution: {str(e)}")
    raise e
finally:
    logger.info(f"Writing all_submissions json")
    with open('all_submissions.json', 'w') as f:
        json.dump(all_submissions, f)
