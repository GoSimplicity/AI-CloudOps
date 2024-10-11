from langchain.llms import Ollama

ollama = Ollama(base_url='http://localhost:11434',
                model="llama3.2:3b", type="json")
print(ollama("why is the sky blue"))
