#!/usr/bin/env python3

# Copyright (c) 2021 RAIL LAB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License in the file LICENSE.txt or at
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""
An example of controlling two robots from within the same routine.

Each robot requires its own device to control it.
"""

#  Used for Cozmo setup
import asyncio
import cozmo
from cozmo.util import degrees, Pose
import cozmo.lights
import sys
#  Used for communication with the server
import json
import urllib.request
import urllib.parse
import math  # Used for calculating waypoints for robots
import unittest  # Used for testing purposes
import time # Time used for wait

#  Globals Describing Team Composition
TEAMS = ['RedTeam', 'BlueTeam']
BOT_NAMES = []
ROBOT_OBJ = []

#  Servers Describing Server Config
SERVER = "http://1f82-2610-148-1f02-3000-b082-3101-7d86-202.ngrok.io"
SERVER_GET = SERVER + "/get"
SERVER_PUT = SERVER + "/put"

#  Testing globals for unittest
TEST_NUM_ROBOTS = 3
TESTING = False
RUN = True

#  Global Color Variables
# Red
RED_TEAM_COLOR = cozmo.lights.Light(cozmo.lights.Color(rgb=(150, 0, 0)))
RED_TEAM_RED_FLAG = cozmo.lights.Light(cozmo.lights.Color(rgb=(255, 0, 0)))
RED_TEAM_BLUE_FLAG = cozmo.lights.Light(cozmo.lights.Color(rgb=(255, 0, 255)))
# Blue
BLUE_TEAM_COLOR = cozmo.lights.Light(cozmo.lights.Color(rgb=(0, 0, 150)))
BLUE_TEAM_BLUE_FLAG = cozmo.lights.Light(cozmo.lights.Color(rgb=(0, 0, 255)))
BLUE_TEAM_RED_FLAG = cozmo.lights.Light(cozmo.lights.Color(rgb=(0, 255, 255)))


# Enumerates basic setup functions for the cozmo including logging
#
#
#
def setup_cozmos():
    # Set up cozmo by getting an event object and setting up logging
    cozmo.setup_basic_logging()
    loop = asyncio.get_event_loop()
    return loop


# Setup for multi-robot connection-
# function checks to see if communication is possible with each of the specified number of robots
#
# \param num_robots =  the number of robots that you want to connect to
#
# \returns a list of robot objects
#
def enumerate_robot_conn(event_loop):
    #  Helpful globals
    global BOT_NAMES
    global TEAMS
    global SERVER_GET
    global ROBOT_OBJ

    #  Get the data from the sever and initialize team names, bot names and number of robots
    with urllib.request.urlopen(SERVER_GET) as url:
        data = json.loads(url.read().decode())
    for team in TEAMS:
        BOT_NAMES.extend(list(data[team].keys()))
    num_robots = len(BOT_NAMES)

    #  Connect to each of the robots
    robot_con_list = []
    for i in range(num_robots):
        trying_to_connect_again = True
        while(trying_to_connect_again):
            try:
                robot_con_list.append(cozmo.connect_on_loop(event_loop))
                trying_to_connect_again = False
            except cozmo.ConnectionError as e:
                print("A connection error occurred: %s" % e)
                print("Will try to connect again in 2 seconds")
                time.sleep(2)


    return robot_con_list


# A function that returns the current locations of all robots
#
# \param a list of robot connection objects that is used to fetch their poses
#
def get_robot_pose(robot_con_list):
    global ROBOT_OBJ
    #  Get all the robot poses
    poses = []
    robots = []
    loop = asyncio.get_event_loop()
    if ROBOT_OBJ == []:
        for robot_con in robot_con_list:
            robots.append(loop.run_until_complete(robot_con.wait_for_robot()))
    else:
        robots = ROBOT_OBJ
    ROBOT_OBJ = robots
    for robot in robots:
        poses.append([robot.pose.position.x, robot.pose.position.y])

    return poses


# A function that checks a sever for poses and returns a list of poses
#
def get_waypoints():
    #  Helpful globals an initialization
    global BOT_NAMES
    global SERVER_GET
    waypoints = []
    should_move = []

    #  Load the data from server
    with urllib.request.urlopen(SERVER_GET) as url:
        data = json.loads(url.read().decode())

    #  Put the data from the server into a list and create the list of waypoints and move booleans
    bot_info = {}
    for team in TEAMS:
        bot_info.update(data[team])
    #BOT_NAMES.extend(list(bot_info.keys()))
    for bot in BOT_NAMES:
        waypoints.append(bot_info[bot]['Waypoint'])
        should_move.append(bot_info[bot]['CanMove'])

    return waypoints, should_move


# A function edits converts the final waypoint and turns it into a sub-waypoint
#
# \param a list of robot connection objects that is used to fetch their poses
#
def edit_waypoints(current_poses, waypoints, should_move_list):
    edited_waypoints = []
    for current_pose, waypoint, should_move in zip(current_poses, waypoints, should_move_list):
        if should_move:
            # calc the hypotenuse and make a singe step waypoint
            waypoint_hyp = math.sqrt((current_pose[0] - waypoint[0]) ** 2 + (current_pose[1] - waypoint[1]) ** 2)
            if(waypoint_hyp):
                new_x = waypoint[0]#current_pose[0] - float(current_pose[0] - waypoint[0])/(waypoint_hyp)
                new_y = waypoint[1]#current_pose[1] - float(current_pose[1] - waypoint[1])/(waypoint_hyp)
            else:
                new_x = float(current_pose[0])
                new_y = float(current_pose[1])
            edited_waypoints.append([new_x, new_y])
        else:
            # if robots cant move make current waypoint be its location
            edited_waypoints.append([current_pose[0], current_pose[1]])

    return edited_waypoints


# A function that takes a list of robot connection objects and poses and bring the robots to those poses
#
# \param a list of robot connection objects that is used to send their waypoints
# \param a list of waypoints
#
async def move_robots_to_waypoint(robot_con_list, waypoint_list):
    global  ROBOT_OBJ
    robots = []

    if ROBOT_OBJ == []:
        for robot_con in robot_con_list:
            robots.append(await robot_con.wait_for_robot())
        ROBOT_OBJ = robots
    else:
         robots = ROBOT_OBJ

    movements = []
    for robot, waypoint in zip(robots, waypoint_list):
        move = robot.go_to_pose(pose=Pose(waypoint[0]*600, waypoint[1]*600, 0, angle_z=degrees(0)), relative_to_robot=False, in_parallel=True)
        robot.set_head_angle(cozmo.util.Angle(degrees=0), in_parallel=True)

    #for move in movements:
    #    await move.wait_for_completed()


# A function that sends to a server the poses of each robot in a team
#
# \param a list of robot connection objects that is used to send their poses
#
def send_poses(robot_con_list):
    #  Helpful globals
    global BOT_NAMES
    global TEAMS
    global SERVER_GET
    global SERVER_PUT
    global ROBOT_OBJ

    # get the robots poses and add them to a json object to be returned
    poses = get_robot_pose(robot_con_list=robot_con_list)

    #  Load the data from server
    with urllib.request.urlopen(SERVER_GET) as url:
        data = json.loads(url.read().decode())

    for bot, pose in zip(BOT_NAMES, poses):
        put = {}
        if bot in data[TEAMS[0]].keys():
            x = pose[0]
            y = pose[1]
            put["Team"] = 'RedTeam'
            put['Value'] = [x/600, y/600]
        elif bot in data[TEAMS[1]].keys():
            x = pose[0]
            y = pose[1]
            put["Team"] = 'BlueTeam'
            put['Value'] = [x/600, y/600]
        #  send the constructed json object
        put['Field'] = 'Location'
        put['OculusId'] = 0
        put['Robot'] = bot
        put = urllib.parse.urlencode(put).encode()
        req = urllib.request.Request(SERVER_PUT, put)
        urllib.request.urlopen(req)


# A function that lights a cozmos led to determine its team and flag
#
# \param a list of robot connection objects that is used to change their leds
#
def light_led(robot_con_list):
    # helpful globals
    global SERVER_GET
    global BOT_NAMES
    global RED_TEAM_COLOR
    global RED_TEAM_RED_FLAG
    global RED_TEAM_BLUE_FLAG
    global BLUE_TEAM_COLOR
    global BLUE_TEAM_BLUE_FLAG
    global BLUE_TEAM_RED_FLAG
    global ROBOT_OBJ
    # get information from the server
    with urllib.request.urlopen(SERVER_GET) as url:
        data = json.loads(url.read().decode())

    # light the robots up with their team color and an indicator of if they are holding the flag
    robots = []
    loop = asyncio.get_event_loop()
    if ROBOT_OBJ == []:
        for robot_con in robot_con_list:
            robots.append(loop.run_until_complete(robot_con.wait_for_robot()))
        ROBOT_OBJ = robots
    else:
        robots = ROBOT_OBJ


    for robot, bot in zip(robots, BOT_NAMES):
        if bot in data[TEAMS[0]].keys():
            if data[TEAMS[0]][bot]['HasBlueFlag']:
                robot.set_all_backpack_lights(RED_TEAM_BLUE_FLAG)
            elif data[TEAMS[0]][bot]['HasRedFlag']:
                robot.set_all_backpack_lights(RED_TEAM_RED_FLAG)
            else:
                robot.set_all_backpack_lights(RED_TEAM_COLOR)
        if bot in data[TEAMS[1]].keys():
            if data[TEAMS[1]][bot]['HasBlueFlag']:
                robot.set_all_backpack_lights(BLUE_TEAM_BLUE_FLAG)
            elif data[TEAMS[1]][bot]['HasRedFlag']:
                robot.set_all_backpack_lights(BLUE_TEAM_RED_FLAG)
            else:
                robot.set_all_backpack_lights(BLUE_TEAM_COLOR)


# A function runs the robots through a collection of waypoints pulled from a server
#
#
def wrangle_robots():
    event_loop = setup_cozmos()  # set up all needed cozmo functions
    robot_cons = enumerate_robot_conn(event_loop)  # create a list containing the information needed to communicate with all the cozmos
    send_poses(robot_con_list=robot_cons)  # move the robots towards their goal location
    waypoints = []  # initially use an empty waypoints, so it can be compared for prior_waypoints
    while True:
        current_poses = get_robot_pose(robot_con_list=robot_cons)  # get a list of the robots current locations
        prior_waypoints = [[x for x in y] for y in waypoints]  # copy the prior waypoints
        waypoints, should_move_list = get_waypoints()  # gets the poses from the server
        print("Waypoints", waypoints)

        if waypoints != prior_waypoints:  # if the waypoints were updated send them to the robots
            edited_poses = edit_waypoints(current_poses=current_poses, waypoints=waypoints, should_move_list=should_move_list)  # gives a partial waypoint on the way to final goal waypoint
            event_loop.run_until_complete(move_robots_to_waypoint(robot_con_list=robot_cons, waypoint_list=edited_poses))  # Move the robots to a specified pose

        light_led(robot_con_list=robot_cons)  # change the robot leds to show team and flag position
        print("Sending position")
        send_poses(robot_con_list=robot_cons)  # move the robots towards their goal location


class TestRoboWrangler(unittest.TestCase):
    # Test that the setup and enumerate robot function works as intended
    def test_Enumerate(self):
        global TEST_NUM_ROBOTS
        loop = setup_cozmos()
        robot_con_list = enumerate_robot_conn(loop)
        print("Found " + str(len(robot_con_list)) + " robots")
        self.assertTrue(len(robot_con_list), TEST_NUM_ROBOTS)

    # Test that the get waypoints and edit waypoints functions work
    def test_get_waypoints_and_edit(self):
        loop = setup_cozmos()
        robot_cons = enumerate_robot_conn(loop)
        waypoints, should_move_list = get_waypoints()  # gets the poses from the server
        current_poses = get_robot_pose(robot_con_list=robot_cons)  # get a list of the robots current locations
        edited_waypoints = edit_waypoints(current_poses=current_poses, waypoints=waypoints, should_move_list=should_move_list)  #
        self.assertIsNotNone(edited_waypoints)

    # test that the send poses function works
    def test_send(self):
        loop = setup_cozmos()
        robot_cons = enumerate_robot_conn(loop)
        true_pose = get_robot_pose(robot_con_list=robot_cons)
        send_poses(robot_con_list=robot_cons)  # move the robots towards their goal location
        server_poses = get_robot_pose(robot_con_list=robot_cons) # get server data
        #self.assertEqual(true_pose, server_poses)

    # test that we can actually move the robots
    def test_robot_move(self):
        global TEST_NUM_ROBOTS
        loop = setup_cozmos()
        robot_cons = enumerate_robot_conn(loop)
        pose_list = []
        for i in range(TEST_NUM_ROBOTS):
            pose_list.append([20, 0])
        move_robots_to_waypoint(robot_cons, pose_list)

    # test that we can actually set the robot LEDS
    def test_robot_led(self):
        loop = setup_cozmos()
        robot_cons = enumerate_robot_conn(loop)
        light_led(robot_cons)
        while(True):
            hi = 1

    # test that  the entire system works
    def test_run(self):
        wrangle_robots()


if __name__ == '__main__':
    # RUn the full code
    if RUN:
        wrangle_robots()
    # Unittest that confirm everything is working as expected
    elif TESTING:
        unittest.main()
    # Debug code
    else:
        loop = setup_cozmos()
        bots = enumerate_robot_conn(loop)
        while(True):
            get_waypoints()
            time.sleep(5)

        print(waypoints)
