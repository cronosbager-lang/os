"""Vector store for semantic memory."""

from typing import List, Dict, Any, Optional, Tuple
from dataclasses import dataclass, field
import json
import os
import math
import hashlib
from pathlib import Path


@dataclass
class Document:
    """Represents a document in the vector store."""
    id: str
    content: str
    embedding: List[float] = field(default_factory=list)
    metadata: Dict[str, Any] = field(default_factory=dict)
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "id": self.id,
            "content": self.content,
            "embedding": self.embedding,
            "metadata": self.metadata,
        }
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "Document":
        return cls(
            id=data["id"],
            content=data["content"],
            embedding=data.get("embedding", []),
            metadata=data.get("metadata", {}),
        )


class SimpleEmbedder:
    """Simple bag-of-words embedder (no external dependencies)."""
    
    def __init__(self, vocab_size: int = 1000):
        self.vocab_size = vocab_size
        self.vocab: Dict[str, int] = {}
    
    def _tokenize(self, text: str) -> List[str]:
        """Simple tokenization."""
        # Lowercase and split on non-alphanumeric
        import re
        tokens = re.findall(r'\b\w+\b', text.lower())
        return tokens
    
    def _hash_token(self, token: str) -> int:
        """Hash token to vocab index."""
        return int(hashlib.md5(token.encode()).hexdigest(), 16) % self.vocab_size
    
    def embed(self, text: str) -> List[float]:
        """Create embedding for text."""
        tokens = self._tokenize(text)
        
        # Create sparse vector
        vector = [0.0] * self.vocab_size
        
        for token in tokens:
            idx = self._hash_token(token)
            vector[idx] += 1.0
        
        # Normalize
        magnitude = math.sqrt(sum(v * v for v in vector))
        if magnitude > 0:
            vector = [v / magnitude for v in vector]
        
        return vector
    
    def embed_batch(self, texts: List[str]) -> List[List[float]]:
        """Create embeddings for multiple texts."""
        return [self.embed(text) for text in texts]


class VectorStore:
    """Simple vector store for semantic search."""
    
    def __init__(
        self,
        path: str = None,
        embedder: SimpleEmbedder = None,
    ):
        self.path = path
        self.embedder = embedder or SimpleEmbedder()
        self.documents: Dict[str, Document] = {}
        
        if path and os.path.exists(path):
            self.load()
    
    def add(
        self,
        content: str,
        metadata: Dict[str, Any] = None,
        doc_id: str = None,
    ) -> str:
        """Add a document to the store."""
        if doc_id is None:
            doc_id = hashlib.sha256(content.encode()).hexdigest()[:16]
        
        embedding = self.embedder.embed(content)
        
        doc = Document(
            id=doc_id,
            content=content,
            embedding=embedding,
            metadata=metadata or {},
        )
        
        self.documents[doc_id] = doc
        return doc_id
    
    def add_batch(
        self,
        contents: List[str],
        metadatas: List[Dict[str, Any]] = None,
    ) -> List[str]:
        """Add multiple documents."""
        if metadatas is None:
            metadatas = [{}] * len(contents)
        
        ids = []
        for content, metadata in zip(contents, metadatas):
            doc_id = self.add(content, metadata)
            ids.append(doc_id)
        
        return ids
    
    def search(
        self,
        query: str,
        k: int = 5,
        threshold: float = 0.0,
    ) -> List[Tuple[Document, float]]:
        """Search for similar documents."""
        query_embedding = self.embedder.embed(query)
        
        results = []
        for doc in self.documents.values():
            similarity = self._cosine_similarity(query_embedding, doc.embedding)
            if similarity >= threshold:
                results.append((doc, similarity))
        
        # Sort by similarity (descending)
        results.sort(key=lambda x: x[1], reverse=True)
        
        return results[:k]
    
    def _cosine_similarity(self, a: List[float], b: List[float]) -> float:
        """Calculate cosine similarity between two vectors."""
        if len(a) != len(b):
            return 0.0
        
        dot_product = sum(x * y for x, y in zip(a, b))
        magnitude_a = math.sqrt(sum(x * x for x in a))
        magnitude_b = math.sqrt(sum(x * x for x in b))
        
        if magnitude_a == 0 or magnitude_b == 0:
            return 0.0
        
        return dot_product / (magnitude_a * magnitude_b)
    
    def get(self, doc_id: str) -> Optional[Document]:
        """Get a document by ID."""
        return self.documents.get(doc_id)
    
    def delete(self, doc_id: str) -> bool:
        """Delete a document."""
        if doc_id in self.documents:
            del self.documents[doc_id]
            return True
        return False
    
    def clear(self):
        """Clear all documents."""
        self.documents.clear()
    
    def count(self) -> int:
        """Get document count."""
        return len(self.documents)
    
    def save(self, path: str = None):
        """Save store to disk."""
        path = path or self.path
        if not path:
            raise ValueError("No path specified")
        
        os.makedirs(os.path.dirname(path), exist_ok=True)
        
        data = {
            "documents": [doc.to_dict() for doc in self.documents.values()],
        }
        
        with open(path, "w") as f:
            json.dump(data, f)
    
    def load(self, path: str = None):
        """Load store from disk."""
        path = path or self.path
        if not path or not os.path.exists(path):
            return
        
        with open(path) as f:
            data = json.load(f)
        
        self.documents.clear()
        for doc_data in data.get("documents", []):
            doc = Document.from_dict(doc_data)
            self.documents[doc.id] = doc


class MemoryIndex:
    """Index for long-term memory with semantic search."""
    
    def __init__(self, path: str = None):
        self.path = path or os.path.expanduser("~/.mixos/agent/memory")
        os.makedirs(self.path, exist_ok=True)
        
        self.conversations = VectorStore(
            path=os.path.join(self.path, "conversations.json")
        )
        self.facts = VectorStore(
            path=os.path.join(self.path, "facts.json")
        )
        self.commands = VectorStore(
            path=os.path.join(self.path, "commands.json")
        )
    
    def remember_conversation(
        self,
        user_message: str,
        assistant_response: str,
        session_id: str = None,
    ):
        """Remember a conversation exchange."""
        content = f"User: {user_message}\nAssistant: {assistant_response}"
        metadata = {
            "type": "conversation",
            "session_id": session_id,
            "timestamp": __import__("time").time(),
        }
        self.conversations.add(content, metadata)
    
    def remember_fact(self, fact: str, source: str = None):
        """Remember a fact."""
        metadata = {
            "type": "fact",
            "source": source,
            "timestamp": __import__("time").time(),
        }
        self.facts.add(fact, metadata)
    
    def remember_command(
        self,
        command: str,
        result: str,
        success: bool = True,
    ):
        """Remember a command execution."""
        content = f"Command: {command}\nResult: {result}"
        metadata = {
            "type": "command",
            "success": success,
            "timestamp": __import__("time").time(),
        }
        self.commands.add(content, metadata)
    
    def recall(self, query: str, k: int = 5) -> List[Dict[str, Any]]:
        """Recall relevant memories."""
        results = []
        
        # Search all stores
        for store_name, store in [
            ("conversations", self.conversations),
            ("facts", self.facts),
            ("commands", self.commands),
        ]:
            for doc, score in store.search(query, k=k):
                results.append({
                    "content": doc.content,
                    "type": store_name,
                    "score": score,
                    "metadata": doc.metadata,
                })
        
        # Sort by score
        results.sort(key=lambda x: x["score"], reverse=True)
        
        return results[:k]
    
    def save(self):
        """Save all stores."""
        self.conversations.save()
        self.facts.save()
        self.commands.save()
    
    def load(self):
        """Load all stores."""
        self.conversations.load()
        self.facts.load()
        self.commands.load()
