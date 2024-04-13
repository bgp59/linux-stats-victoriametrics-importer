from base64 import b64encode
from codecs import utf_8_decode


def b64encode_str(s: str) -> str:
    return utf_8_decode(b64encode(bytes(s, "utf-8")))[0]
