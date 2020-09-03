from tomlkit.toml_file import TOMLFile
from tomlkit.items import Table as TomlTable
from tomlkit.items import Array as TomlArray
from pathlib import Path
from typing import Dict, Any, Text, List, Mapping, Set
from copy import deepcopy
from dataclasses import dataclass
import logging
import click


LOG = logging.getLogger(__name__)


class Borg:
    """Classic Borg."""
    _shared_state = {}

    def __init__(self):
        self.__dict__ = self._shared_state


@dataclass
class State:
    ssh_keys: Mapping[Text, Mapping[Text, Any]] = None
    toml_file: TOMLFile = None
    toml_doc: Any = None


@dataclass
class Config:
    toml_file: TOMLFile = None
    toml_doc: Any = None


@dataclass
class Context:
    # Click
    state_file: Path
    config_file: Path
    ssh_dir: Path

    key_groups: Dict[Text, List[Text]] = None

    state: State = None
    config: Config = None
    session: Any = None


class ContextBorg(Context, Borg):
    def __init__(self, init_data: Mapping = None):
        Borg.__init__(self)
        if init_data:
            Context.__init__(self, **init_data)  # noqa ("**init_data": "unexpected argument")
            self.read_state()
            self.read_config()
            self.consume_groups()

    def __str__(self):
        return f"ContextBorg({id(self)})"

    def __repr__(self):
        return f"ContextBorg({id(self)})"

    def read_state(self):
        if not self.state:
            self.state = State()
        self.state.toml_file = TOMLFile(self.state_file.as_posix())
        self.state.toml_doc = self.state.toml_file.read()

    def read_config(self):
        if not self.config:
            self.config = Config()
        self.config.toml_file = TOMLFile(self.config_file.as_posix())
        self.config.toml_doc = self.config.toml_file.read()

    def consume_groups(self):
        if not self.key_groups:
            self.key_groups = dict()

        for lvl_1_k, lvl_1_v in self.config.toml_doc["key_groups"].items():
            if isinstance(lvl_1_v, dict):  # Note: this does not return TomlTable like you'd expect.
                LOG.debug(f"Found meta-group: {lvl_1_k}")
                for lvl_2_k, lvl_2_v in lvl_1_v.items():
                    combo_key = f"{lvl_1_k}.{lvl_2_k}"
                    LOG.debug(f"Found group: {combo_key}")
                    self.key_groups[combo_key] = lvl_2_v.copy()

            elif isinstance(lvl_1_v, TomlArray):
                LOG.debug(f"Found group: {lvl_1_k}")
                self.key_groups[lvl_1_k] = lvl_1_v.copy()

            else:
                LOG.error(f"Unexpected type found in config toml: Found: {type(lvl_1_v)}")
                raise click.Abort


def write_ssh_key(key_file_path: Path, input_data: Dict[Text, Any]) -> None:
    """Save SSH key to state file."""
    c = ContextBorg()

    if key_file_path.name in c.state.toml_doc["ssh_key"].keys():
        LOG.warning(f'Overwriting SSH key entry that already exists for "{key_file_path.name}"')

    c.state.toml_doc["ssh_key"][key_file_path.name] = deepcopy(input_data)
    c.state.toml_file.write(c.state.toml_doc)


def get_key_list(state_path: Path) -> List[Any]:
    c = ContextBorg()
    c.toml_file = TOMLFile(state_path.as_posix())
    c.toml_doc = c.toml_file.read()

    ret = []
    for x in c.toml_doc["ssh_key"].keys():
        ret.append(x)
    return ret
