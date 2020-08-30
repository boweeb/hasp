from tomlkit.toml_file import TOMLFile
from pathlib import Path
from typing import Dict, Any, Text, List, Mapping
from copy import deepcopy
from dataclasses import dataclass
import logging


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
    foo: Text


@dataclass
class Context:
    # Click
    state_file: Path
    config_file: Path
    ssh_dir: Path

    state: State
    config: Config


class ContextBorg(Context, Borg):
    def __init__(self, init_data: Mapping = None):
        Borg.__init__(self)
        if init_data:
            Context.__init__(self, **init_data)  # noqa ("**init_data": "unexpected argument")
        self.read_state()

    def __str__(self):
        return f"ContextBorg({id(self)})"

    def __repr__(self):
        return f"ContextBorg({id(self)})"

    def read_state(self):
        if not self.state:
            self.state = State()
        self.state.toml_file = TOMLFile(self.state_file.as_posix())
        self.state.toml_doc = self.state.toml_file.read()


def write_ssh_key(key_file_path: Path, input_data: Dict[Text, Any]) -> None:
    """Save SSH key to state file.

    Args:
        key_file_path:
        input_data:
        state_path:

    Returns:

    """
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
