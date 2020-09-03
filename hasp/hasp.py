import logging
import re
from copy import deepcopy
from pathlib import Path
from subprocess import run
from typing import List, Text, Any, Optional, Mapping, Union

import magic
from tomlkit import table as toml_table

from hasp.io import ContextBorg
from hasp.orm import SshKey, SshKeyGroup, SshKeyAlias


LOG = logging.getLogger(__name__)


def present_key_list() -> None:
    c = ContextBorg()
    key_list: List[Any] = c.state.toml_doc["ssh_key"].keys()
    for key in key_list:
        present_key(key)


def present_key(key) -> None:
    c = ContextBorg()

    key_map = {}
    for k, v in c.state.toml_doc["ssh_key"][key].items():
        key_map[k] = v

    key_copy = deepcopy(key_map)

    print(key)

    # print("\t", end="")
    # pprint(c.state.toml_doc["ssh_key"][key])
    # rprint(key_copy)
    for k, v in key_copy.items():
        print(f'\t{k:>8}: "{v}"')
    print()


def sync_keys(name: Optional[Text], orientation: Optional[Text]) -> None:
    LOG.debug(f"{name=}")
    LOG.debug(f"{orientation=}")

    c = ContextBorg()

    init = False
    if orientation in ["init", "both", None]:
        init = True

    prune = False
    if orientation in ["prune", "both", None]:
        prune = True

    # if init:
    #     LOG.info("Begin SSH key initialization")
    #     if name:
    #         init_state_entries(c.ssh_dir / name)
    #     else:
    #         for key_file in c.ssh_dir.glob("*"):
    #             init_state_entries(key_file)
    #
    # if prune:
    #     LOG.info("Begin SSH key pruning")
    #     if name:
    #         prune_state_entries(name)
    #     else:
    #         for key in resolve_toml_keys(c.state.toml_doc["ssh_key"]):
    #             prune_state_entries(key)

    init_state_groups()
    prune_state_groups()


def prune_state_entries(key_name: Text):
    c = ContextBorg()
    key_map: toml_table = c.state.toml_doc["ssh_key"]

    resolved_key_list = resolve_toml_keys(key_map)

    if key_name in resolved_key_list:
        if not Path(c.ssh_dir / key_name).exists():
            LOG.info(f'Pruning stale SSH key entry in state: "{key_name}"')
            key_map.remove(key_name)
            # commit_toml()
            c.state.toml_file.write(c.state.toml_doc)
    else:
        LOG.info(f'Key does is not in state: "{key_name}"')


def resolve_toml_keys(element) -> List[Text]:
    # Resolve generator into actual list
    resolved_list = []
    for k in element.keys():
        resolved_list.append(k)
    return resolved_list


def read_real_key(key_file: Path):
    # c = ContextBorg()

    magic_type = magic.from_file(key_file.as_posix())
    if re.match(r"(?i)^(OpenSSH|PEM).*private.*\bkey$", magic_type):
        LOG.info(f'Found private key: "{key_file.name}"')
        cmd = ["ssh-keygen", "-l", "-E", "md5", "-f", key_file]
        result = run(cmd, text=True, capture_output=True, check=True)
        lines = result.stdout.rstrip().split()

        bits = int(lines.pop(0))
        hash_base = lines.pop(0).split(":")
        _hash_md5 = hash_base.pop(0)
        hash_value = ":".join(hash_base)
        type_ = lines.pop(-1).lstrip('(').rstrip(')').lower()
        comment = " ".join(lines)

        real_key = {
            "type": type_,
            "comment": comment,
            "bits": bits,
            "md5": hash_value,
            "format": magic_type.split(" ")[0].lower(),
        }

        return real_key


def init_state_entries(key_file):
    # c = ContextBorg()

    if key_file.is_file() and not key_file.is_symlink():
        real_key = read_real_key(key_file)
        update_key(key_file, real_key)
    elif key_file.is_file() and key_file.is_symlink():
        target = key_file.resolve()
        LOG.info(f'Found symlink: "{key_file.name}" --> "{target.name}"')
        update_key(target, aliases=[key_file])
    else:
        LOG.debug(f'SKIPPING: Not a file: "{key_file}"')
        # raise click.Abort


def init_state_groups() -> None:
    c = ContextBorg()

    for group_, members in c.key_groups.items():
        group: SshKeyGroup = c.session.query(SshKeyGroup).filter(SshKeyGroup.name == group_).one_or_none()
        if not group:
            group = SshKeyGroup(name=group_)
            LOG.info(f'Adding missing group: "{group.name}"')
            c.session.add(group)

        for member in members:
            ssh_key: SshKey = c.session.query(SshKey).filter(SshKey.name == member).one_or_none()
            if ssh_key:
                group.ssh_keys.append(ssh_key)
            else:
                LOG.info(f'Found host member in config file that has no counterpart: "{group_}/{member}"')

    c.session.commit()


def prune_state_groups():
    pass


def update_key(
    key_file: Path,
    data: Optional[Mapping[Text, Union[Text, int]]] = None,
    aliases: Optional[List[Path]] = None
):
    c = ContextBorg()

    state_key_query = c.session.query(SshKey).order_by(SshKey.name)
    state_key_list = state_key_query.all()
    resolved_key_list: List[Text] = [k.name for k in state_key_list]

    if key_file.name in resolved_key_list:
        state_key: SshKey = state_key_query.filter(SshKey.name == key_file.name).one()

        LOG.debug(f"{state_key=}")
        LOG.debug(f"{data=}")

        if data:
            for attr in data.keys():

                if not hasattr(state_key, attr):
                    _state_attr = None
                else:
                    _state_attr = getattr(state_key, attr)

                if not _state_attr:
                    print(f"{attr:<10}:            <NEW> --> {data[attr]}")
                elif _state_attr != data[attr]:
                    print(f'{attr:<10}: {_state_attr:>16} --> {data[attr]}')

                setattr(state_key, attr, data[attr])

        if aliases:
            # resolved_list = resolve_toml_keys(state_key)

            for alias in aliases:
                # if "aliases" not in resolved_list:
                #     state_key["aliases"] = []

                resolved_alias_list = []
                for existing_alias in state_key.ssh_key_aliases:
                    resolved_alias_list.append(existing_alias.name)

                if alias.name not in resolved_alias_list:
                    state_key_alias = SshKeyAlias(name=alias.name)
                    state_key.ssh_key_aliases.append(state_key_alias)

                # alias_set = set(state_key["aliases"])
                # alias_set.add(alias.name)
                # state_key["aliases"] = list(alias_set)

    else:
        # "if aliases" does not apply here.
        if data:
            LOG.info("Importing new SSH key entry into state")
            new_key = SshKey(name=key_file.name, **data)
            c.session.add(new_key)
            c.session.commit()
            # key_map[key_file.name] = deepcopy(data)

    # TODO: Left off here
    # key_group_list: List[Text] = resolve_toml_keys(c.config.toml_doc["key_groups"])

    c.session.commit()
    # c.state.toml_file.write(c.state.toml_doc)
    # commit_toml()


# def update_key(
#     key_file: Path,
#     data: Optional[Mapping[Text, Union[Text, int]]] = None,
#     aliases: Optional[List[Path]] = None
# ):
#     c = ContextBorg()
#
#     key_map = c.state.toml_doc["ssh_key"]
#     resolved_key_list = resolve_toml_keys(key_map)
#
#     if key_file.name in resolved_key_list:
#         state_key = key_map[key_file.name]
#
#         LOG.debug(f"{state_key=}")
#         LOG.debug(f"{data=}")
#
#         if data:
#             for attr in ["type", "bits", "comment", "md5", "format"]:
#
#                 if attr not in state_key.keys():
#                     print(f"{attr:<10}:            <NEW> --> {data[attr]}")
#                 elif state_key[attr] != data[attr]:
#                     print(f'{attr:<10}: {state_key[attr]:>16} --> {data[attr]}')
#
#                 state_key[attr] = data[attr]
#
#         if aliases:
#             resolved_list = resolve_toml_keys(state_key)
#
#             for alias in aliases:
#                 if "aliases" not in resolved_list:
#                     state_key["aliases"] = []
#
#                 alias_set = set(state_key["aliases"])
#                 alias_set.add(alias.name)
#                 state_key["aliases"] = list(alias_set)
#
#     else:
#         # "if aliases" does not apply here.
#         if data:
#             LOG.info("Importing new SSH key entry into state")
#             key_map[key_file.name] = deepcopy(data)
#
#     c.state.toml_file.write(c.state.toml_doc)
#     # commit_toml()
#
#
def commit_toml() -> None:
    c = ContextBorg()
    c.state.toml_file.write(c.state.toml_doc)
