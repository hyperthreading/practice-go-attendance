import subprocess
import traceback
import requests
import os
import time


class bcolors:
    HEADER = "\033[95m"
    OKBLUE = "\033[94m"
    OKCYAN = "\033[96m"
    OKGREEN = "\033[92m"
    WARNING = "\033[93m"
    FAIL = "\033[91m"
    ENDC = "\033[0m"
    BOLD = "\033[1m"
    UNDERLINE = "\033[4m"


def run_process(cmd, capture_output=True):
    process = subprocess.Popen(
        cmd,
        stdout=subprocess.PIPE if capture_output else None,
        stderr=subprocess.PIPE if capture_output else None,
        shell=True,
    )
    return process


def wait_process(process: subprocess.Popen, raise_on_error=True):
    out, err = process.communicate()
    if raise_on_error and process.returncode != 0:
        raise Exception(f"Process failed with return code {process.returncode}")
    return out, err


def run_watch_process(context):
    # check if the container is already running
    watch_lock_path = os.path.join(os.path.dirname(__file__), "watch.lock")
    running = False
    with open(watch_lock_path, "a+") as f:
        f.seek(0)
        container_id = f.read()

    container_id = container_id.strip()
    if container_id:
        cmd = f"docker inspect {container_id}"
        process = run_process(cmd)
        wait_process(process, raise_on_error=False)
        running = process.returncode == 0

    if not running:
        cmd = "docker build -t go-builder -f ./build/test/builder.Dockerfile ."
        wait_process(run_process(cmd))

        cmd = "docker run -q -d --rm --name go-builder -v $(pwd):/app go-builder"
        container_id = wait_process(run_process(cmd))[0].decode("utf-8").strip()

        with open(watch_lock_path, "w") as f:
            f.write(container_id)

    cmd = f"docker logs -f {container_id}"
    process = run_process(cmd)
    context["build_log_process"] = process


def assert_equal(a, b, message=None):
    if a == b:
        return
    raise AssertionError(f"{a} != {b} {message or ''}")


def assert_contains_all(a, b, message=None):
    if all([x in a for x in b]):
        return
    raise AssertionError(f"{a} does not contain all {b} {message or ''}")


def setup(context):
    run_watch_process(context)
    server_process = run_process(
        "docker compose -f ./deploy/test/docker-compose.yaml up --wait -d"
    )
    wait_process(server_process)
    log_process = run_process(
        "docker compose -f ./deploy/test/docker-compose.yaml logs -f",
        capture_output=False,
    )
    context["log_process"] = log_process


def teardown(context):
    cmd = run_process("docker compose -f ./deploy/test/docker-compose.yaml down")
    wait_process(cmd)
    context["log_process"].terminate()
    context["build_log_process"].terminate()


def test():
    context = {}
    setup(context)
    context["server_url"] = "http://localhost:8080"
    running_phase = None
    try:
        for name, fn in globals().items():
            if name.startswith("test_") and callable(fn):
                print(f"{bcolors.HEADER}Running {name}...{bcolors.ENDC}")
                running_phase = name
                fn(context)
                print(f"{bcolors.OKGREEN}PASS: {name}{bcolors.ENDC}")
                running_phase = "Teardown"
                reset_time(context)
                reset_database(context)
    except Exception as e:
        print(f"{bcolors.FAIL}FAIL: {running_phase} failed{bcolors.ENDC}")
        traceback.print_exception(e)
    finally:
        teardown(context)


def fix_time(context, timeInString):
    data = {"time": timeInString}
    response = requests.post(
        context.get("server_url", "") + "/test/fix-time", json=data
    )
    assert response.status_code == 200


def reset_time(context):
    response = requests.post(context.get("server_url", "") + "/test/reset-time")
    assert response.status_code == 200


def reset_database(context):
    response = requests.post(context.get("server_url", "") + "/test/reset-database")
    assert response.status_code == 200


def test_health_check(context):
    response = requests.get(context.get("server_url", "") + "/health")
    assert response.status_code == 200
    assert response.json() == {"message": "ok"}


def attend(context, **kwargs):
    data = dummy_payload.copy()
    data["text"] = "attend"
    data.update(kwargs)
    response = requests.post(
        context.get("server_url", "") + "/command/attend", data=data
    )
    return response


def leave(context, **kwargs):
    data = dummy_payload.copy()
    data["text"] = "leave"
    data.update(kwargs)
    response = requests.post(
        context.get("server_url", "") + "/command/attend", data=data
    )
    return response


def list_users_attended(context, **kwargs):
    data = {
        "tz": "+09:00",
    }
    data.update(kwargs)
    return requests.get(
        context.get("server_url", "") + "/user_list/attended", params=data
    )


dummy_payload = {
    "token": "test-token",
    "team_id": "test-team-id",
    "team_domain": "test-team-domain",
    "channel_id": "test-channel-id",
    "channel_name": "test-channel-name",
    "user_id": "test-user-id",
    "user_name": "test-user-name",
    "command": "/attend",
    "text": "",
    "response_url": "test-response-url",
    "trigger_id": "test-trigger-id",
}


def test_fail_leave_before_attend(context):
    response = leave(context)
    assert_equal(response.status_code, 422)
    assert_equal(response.json()["code"], "not_attended_yet")


def test_success_attend_and_leave(context):
    fix_time(context, "2021-01-01T09:00:00Z")

    response = attend(context)
    assert_equal(response.status_code, 200)

    fix_time(context, "2021-01-01T18:00:00Z")
    response_leave = leave(context)
    assert_equal(response_leave.status_code, 200)


def test_multiple_users_attend(context):
    fix_time(context, "2021-01-01T09:00:00Z")

    response = attend(context, user_id="test-user-id-1")
    assert_equal(response.status_code, 200)

    response = attend(context, user_id="test-user-id-2")
    assert_equal(response.status_code, 200)

    response = attend(context, user_id="test-user-id-3")
    assert_equal(response.status_code, 200)

    fix_time(context, "2021-01-01T18:00:00Z")
    response_leave = leave(context, user_id="test-user-id-1")
    assert_equal(response_leave.status_code, 200)

    response_leave = leave(context, user_id="test-user-id-2")
    assert_equal(response_leave.status_code, 200)

    response_leave = leave(context, user_id="test-user-id-3")
    assert_equal(response_leave.status_code, 200)


def test_list_users_attended(context):
    fix_time(context, "2021-01-01T07:00:00Z")

    response = attend(context, user_id="test-user-id-0", user_name="test-user-name-0")

    fix_time(context, "2021-01-01T09:00:00Z")

    response = attend(context, user_id="test-user-id-1", user_name="test-user-name-1")
    assert_equal(response.status_code, 200)

    response = attend(context, user_id="test-user-id-2", user_name="test-user-name-2")
    assert_equal(response.status_code, 200)

    response = attend(context, user_id="test-user-id-3", user_name="test-user-name-3")
    assert_equal(response.status_code, 200)

    fix_time(context, "2021-01-01T18:00:00Z")
    response_leave = leave(context, user_id="test-user-id-1")
    assert_equal(response_leave.status_code, 200)

    response_list = list_users_attended(context, date="2021-01-01", tz="-09:00")
    assert_equal(response_list.status_code, 200, response_list.json()["message"])
    assert_contains_all(
        response_list.json()["data"],
        [
            # This poor user should be included since he attended before 9:00
            {
                "userId": "test-user-id-0",
                "userName": "test-user-name-0",
                "attendedAt": "2021-01-01T07:00:00Z",
            },
            {
                "userId": "test-user-id-1",
                "userName": "test-user-name-1",
                "attendedAt": "2021-01-01T09:00:00Z",
                "leftAt": "2021-01-01T18:00:00Z",
            },
            {
                "userId": "test-user-id-2",
                "userName": "test-user-name-2",
                "attendedAt": "2021-01-01T09:00:00Z",
            },
            {
                "userId": "test-user-id-3",
                "userName": "test-user-name-3",
                "attendedAt": "2021-01-01T09:00:00Z",
            },
        ],
    )


def test_attend_in_specified_time(context):
    fix_time(context, "2021-01-01T12:00:00Z")

    response = attend(
        context,
        user_id="test-user-id-1",
        user_name="test-user-name-1",
        text="add 10:00~19:00",
    )
    assert_equal(response.status_code, 200, response.json())

    response = attend(
        context,
        user_id="test-user-id-2",
        user_name="test-user-name-2",
        text="add 09:00~19:00",
    )
    assert_equal(response.status_code, 200)

    response = attend(
        context,
        user_id="test-user-id-2",
        user_name="test-user-name-2",
        text="add 2020-12-31 10:00~19:00",
    )
    assert_equal(response.status_code, 200, response.json())

    # check in the attendance list
    response_list = list_users_attended(context, date="2020-12-31")
    assert_equal(response_list.status_code, 200)
    assert_contains_all(
        response_list.json()["data"],
        [
            {
                "userId": "test-user-id-2",
                "userName": "test-user-name-2",
                "attendedAt": "2020-12-31T10:00:00Z",
                "leftAt": "2020-12-31T19:00:00Z",
            },
        ],
    )

    # check in the attendance list
    response_list = list_users_attended(context, date="2021-01-01")
    assert_equal(response_list.status_code, 200)
    assert_contains_all(
        response_list.json()["data"],
        [
            {
                "userId": "test-user-id-2",
                "userName": "test-user-name-2",
                "attendedAt": "2021-01-01T09:00:00Z",
                "leftAt": "2021-01-01T19:00:00Z",
            },
            {
                "userId": "test-user-id-1",
                "userName": "test-user-name-1",
                "attendedAt": "2021-01-01T10:00:00Z",
                "leftAt": "2021-01-01T19:00:00Z",
            },
        ],
    )


if __name__ == "__main__":
    test()
