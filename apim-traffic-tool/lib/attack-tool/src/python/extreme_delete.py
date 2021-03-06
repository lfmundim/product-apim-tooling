# Copyright (c) 2020, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
#
# WSO2 Inc. licenses this file to you under the Apache License,
# Version 2.0 (the "License"); you may not use this file except
# in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied. See the License for the
# specific language governing permissions and limitations
# under the License.

import os
import random
import sys
import time
from datetime import datetime
from multiprocessing.dummy import Pool

import pandas as pd
import requests
import yaml
import numpy as np
from utils import util_methods
from utils.entity_classes import API
from utils.util_methods import generate_random_string
from constants import *
from utils import log


# noinspection PyUnusedLocal
def request_handler(i):
    """
    Handle the requests
    :return: None
    """
    global attack_duration, protocol, host, port, payloads, user_agents, api_list, dataset_path

    up_time = datetime.now() - start_time
    if up_time.seconds < attack_duration:
        random_api = random.choice(api_list)
        context = random_api.context
        version = random_api.version
        resource_path = random.choice(random_api.resources['DELETE'])
        random_user = random_api.single_user
        resource_method = "DELETE"
        accept = content_type = "application/json"

        # sleep the process for a random period of time
        time.sleep(abs(int(np.random.normal() * 10)))

        request_path = "{}://{}:{}/{}/{}/{}".format(protocol, host, port, context, version, resource_path)
        random_user_agent = random.choice(user_agents)
        token = random_user[0]
        ip = random_user[2]
        cookie = random_user[3]
        path_param = generate_random_string(10)
        try:
            response = util_methods.send_simple_request(request_path, resource_method, token, ip, cookie, accept, content_type, random_user_agent, path_params=path_param)
            request_info = "{},{},{},{},{}/{},{},{},{},{},\"{}\",{}".format(datetime.now(), ip, token, resource_method, request_path, path_param, cookie, accept, content_type, ip, random_user_agent,
                                                                            response.status_code
                                                                            )
            util_methods.write_to_file(dataset_path, request_info, "a")
        except requests.exceptions.ConnectionError as e:
            error_code = 521
            request_info = "{},{},{},{},{}/{},{},{},{},{},\"{}\",{}".format(datetime.now(), ip, token, resource_method, request_path, path_param, cookie, accept, content_type, ip, random_user_agent,
                                                                            error_code
                                                                            )
            util_methods.write_to_file(dataset_path, request_info, "a")
            logger.error("Connection Error: {}".format(e))
        except requests.exceptions.RequestException:
            logger.exception("Request Failure")


# Program Execution
if __name__ == '__main__':

    logger = log.set_logger('Extreme_Delete')
    try:
        with open(os.path.abspath(os.path.join(__file__, "../../../../../config/api_details.yaml")), "r") as config_file:
            config = yaml.load(config_file, Loader=yaml.FullLoader)

        with open(os.path.abspath(os.path.join(__file__, "../../../../../config/attack-tool.yaml")), "r") as attack_config_file:
            attack_config = yaml.load(attack_config_file, Loader=yaml.FullLoader)
    except FileNotFoundError as ex:
        logger.error("{}: \'{}\'".format(ex.strerror, ex.filename))
        sys.exit()

    # reading configurations from attack-tool.yaml
    protocol = attack_config[GENERAL_CONFIG][API_HOST][PROTOCOL]
    host = attack_config[GENERAL_CONFIG][API_HOST][IP]
    port = attack_config[GENERAL_CONFIG][API_HOST][PORT]
    attack_duration = attack_config[GENERAL_CONFIG][ATTACK_DURATION]
    payloads = attack_config[GENERAL_CONFIG][PAYLOADS]
    user_agents = attack_config[GENERAL_CONFIG][USER_AGENTS]
    process_count = attack_config[GENERAL_CONFIG][NUMBER_OF_PROCESSES]

    # reading api configuration from api_details.yaml
    apis = config[APIS]

    # reading user data (access token, api name,user ip, and cookie)
    user_details = pd.read_csv(os.path.abspath(os.path.join(__file__, "../../../../traffic-tool/data/scenario/token_ip_cookie.csv")))
    user_details_groups = user_details.groupby(API_NAME)

    # Instantiating API objects which has delete methods and appending them to api_list
    api_list = []
    for api in apis:
        temp = API(protocol, host, port, api[CONTEXT], api[VERSION], api[NAME])
        temp.users = user_details_groups.get_group(temp.name).values.tolist()
        temp.set_single_user()
        for resource in api[resources]:
            temp.add_resource(resource[method], resource[path])
        if 'DELETE' in temp.resources.keys():
            api_list.append(temp)

    start_time = datetime.now()

    # Recording column names in the dataset csv file
    dataset_path = "../../../../../../dataset/attack/extreme_delete.csv"
    util_methods.write_to_file(dataset_path, "Timestamp, Request path, Method,Access Token, IP Address, Cookie, Response Code", "w")

    if len(api_list) == 0:
        logger.error("There are no APIs with DELETE endpoints")
        sys.exit()

    logger.info("Extreme delete attack started")
    util_methods.write_to_file(dataset_path, "timestamp,ip_address,access_token,http_method,invoke_path,cookie,accept,content_type,x_forwarded_for,user_agent,response_code", "w")

    process_pool = Pool(processes=process_count)

    # Executing scenarios until the attack duration elapses
    while True:
        time_elapsed = datetime.now() - start_time
        if time_elapsed.seconds >= attack_duration:
            logger.info("Attack terminated successfully. Time elapsed: {} minutes".format(time_elapsed.seconds / 60.0))
            break
        else:
            process_pool.map(request_handler, range(1000))

    # closes the process pool and wait for the processes to finish
    process_pool.close()
    process_pool.join()
