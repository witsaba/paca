#!/usr/bin/env python3
#
# The proof that the agent of PACA reaches the memory of PACA over the
# stdio MCP transport.
#
# The proof is a round trip and not a heartbeat: a server that answers
# does not prove that the memory works. It spawns the MCP server of the
# store exactly as the agent spawns it — the same binary, the same
# arguments, the same environment — it writes a memory through it, it
# reads that memory back through it, and it removes it afterwards, so the
# proof leaves no memory behind.
#
# The removal is done with the administrative profile of the store: the
# tools of the agent deliberately do not delete, and a proof that leaves
# its own record in the memory of the people is a proof that pollutes.

import asyncio
import json
import os
import sys
import time

from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client

# The binary of the store and the tools of the agent: the same ones the
# configuration of the agent names.
BINARY = "/opt/engram/engram"
AGENT_TOOLS = ["mcp", "--tools=agent"]
ADMIN_TOOLS = ["mcp", "--tools=admin"]


def fail(message):
    print("hermes: %s" % message, file=sys.stderr)
    sys.exit(1)


def server_args(tool_profile):
    """The parameters of the MCP server of the store, the way the agent
    spawns it: the environment of the container, with the state of the
    memory and the project of the monorepository in it."""
    environment = dict(os.environ)
    environment["ENGRAM_DATA_DIR"] = os.environ.get("ENGRAM_DATA_DIR", "/opt/engram-data")
    environment["ENGRAM_PROJECT"] = os.environ.get("ENGRAM_PROJECT", "paca")
    return StdioServerParameters(command=BINARY, args=tool_profile, env=environment)


def payload_of(result, what):
    """The JSON envelope of the answer of a tool of the store."""
    if result.isError:
        fail("the store refuses %s: %s" % (what, result.content[0].text if result.content else "no answer"))
    if not result.content:
        fail("the store answers nothing to %s" % what)
    try:
        return json.loads(result.content[0].text)
    except json.JSONDecodeError:
        fail("the store answers something that is not JSON to %s: %s" % (what, result.content[0].text))


async def with_session(tool_profile, body):
    """Runs the body with a live session of the MCP server of the store."""
    async with stdio_client(server_args(tool_profile)) as (reader, writer):
        async with ClientSession(reader, writer) as session:
            await session.initialize()
            return await body(session)


async def round_trip():
    marker = "pacahermessmoke%d" % int(time.time())

    async def write(session):
        tools = [tool.name for tool in (await session.list_tools()).tools]
        for tool in ("mem_save", "mem_search"):
            if tool not in tools:
                fail("the tools of the agent do not carry %s: %s" % (tool, ", ".join(sorted(tools))))
        saved = payload_of(
            await session.call_tool(
                "mem_save",
                {
                    "title": "paca hermes smoke",
                    "content": "the agent of PACA reaches the memory of PACA over the stdio MCP: %s" % marker,
                    "type": "config",
                },
            ),
            "the write of the proof",
        )
        observation = saved.get("id")
        if not observation:
            fail("the store does not keep the memory of the proof: %s" % json.dumps(saved))
        if saved.get("project") != os.environ.get("ENGRAM_PROJECT", "paca"):
            fail("the memory of the proof does not belong to the project of the monorepository")
        return observation, marker

    observation, marker = await with_session(AGENT_TOOLS, write)

    async def read(session):
        found = payload_of(
            await session.call_tool(
                "mem_search",
                {
                    "query": marker,
                    "project": os.environ.get("ENGRAM_PROJECT", "paca"),
                    "response_format": "compact",
                    "limit": 5,
                },
            ),
            "the read of the proof",
        )
        return marker in json.dumps(found)

    if not await with_session(AGENT_TOOLS, read):
        fail("the store does not give the memory of the proof back")

    async def remove(session):
        deleted = payload_of(
            await session.call_tool(
                "mem_delete",
                {"id": observation, "expected_project": os.environ.get("ENGRAM_PROJECT", "paca"), "hard_delete": True},
            ),
            "the removal of the proof",
        )
        return deleted

    await with_session(ADMIN_TOOLS, remove)
    print("hermes: the memory is written and given back over the stdio MCP")


asyncio.run(round_trip())
