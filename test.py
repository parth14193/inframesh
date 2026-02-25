"""Rank-based LFU cache with LRU tie-breaker for same rank."""

from collections import OrderedDict, defaultdict
from dataclasses import dataclass
import unittest


@dataclass
class CacheItem:
    key: str
    value: str
    rank: int


class LFUCache:
    """
    Eviction policy:
    1) Lowest rank is evicted first.
    2) If multiple keys share the same rank, evict the least recently used key.
    """

    def __init__(self, capacity: int):
        self.capacity = capacity
        self.cache: dict[str, CacheItem] = {}
        self.rank_to_keys: dict[int, OrderedDict[str, None]] = defaultdict(OrderedDict)

    def _touch(self, key: str) -> None:
        """Move key to MRU position within its current rank bucket."""
        item = self.cache[key]
        bucket = self.rank_to_keys[item.rank]
        bucket.move_to_end(key, last=True)

    def get(self, key: str):
        if key not in self.cache:
            return None
        self._touch(key)
        return self.cache[key].value

    def put(self, key: str, item: CacheItem) -> None:
        if self.capacity <= 0:
            return

        if key in self.cache:
            old_item = self.cache[key]
            if old_item.rank != item.rank:
                old_bucket = self.rank_to_keys[old_item.rank]
                del old_bucket[key]
                if not old_bucket:
                    del self.rank_to_keys[old_item.rank]
            self.cache[key] = item
            self.rank_to_keys[item.rank][key] = None
            self._touch(key)
            return

        self.cache[key] = item
        self.rank_to_keys[item.rank][key] = None
        if len(self.cache) > self.capacity:
            self._evict()

    def _evict(self) -> None:
        min_rank = min(self.rank_to_keys.keys())
        lru_key, _ = self.rank_to_keys[min_rank].popitem(last=False)
        if not self.rank_to_keys[min_rank]:
            del self.rank_to_keys[min_rank]
        del self.cache[lru_key]


class TestLFUCache(unittest.TestCase):
    def test_evicts_lowest_rank_first(self):
        cache = LFUCache(capacity=2)
        cache.put("a", CacheItem("a", "A", rank=10))
        cache.put("b", CacheItem("b", "B", rank=1))
        cache.put("c", CacheItem("c", "C", rank=5))

        self.assertIsNone(cache.get("b"))
        self.assertEqual(cache.get("a"), "A")
        self.assertEqual(cache.get("c"), "C")

    def test_lru_tie_breaker_with_same_rank(self):
        cache = LFUCache(capacity=2)
        cache.put("a", CacheItem("a", "A", rank=1))
        cache.put("b", CacheItem("b", "B", rank=1))
        cache.get("a")  # a becomes more recent than b
        cache.put("c", CacheItem("c", "C", rank=1))

        self.assertIsNone(cache.get("b"))
        self.assertEqual(cache.get("a"), "A")
        self.assertEqual(cache.get("c"), "C")


if __name__ == "__main__":
    unittest.main()
