#include <iostream>
#include <fstream>
#include <string>
#include <windows.h>

extern "C" {
    typedef char* (*TextMiner_ExtractFileFunc)(const char* filePath, int enableOcr);
}

class TextMinerWrapper {
private:
    HMODULE hDll;
    TextMiner_ExtractFileFunc TextMiner_ExtractFile;

public:
    TextMinerWrapper() : hDll(nullptr), TextMiner_ExtractFile(nullptr) {}

    bool load(const std::string& dllPath) {
        hDll = LoadLibraryA(dllPath.c_str());
        if (!hDll) {
            std::cerr << "Failed to load DLL: " << dllPath << std::endl;
            return false;
        }

        TextMiner_ExtractFile = (TextMiner_ExtractFileFunc)GetProcAddress(hDll, "TextMiner_ExtractFile");
        if (!TextMiner_ExtractFile) {
            std::cerr << "Failed to get function address from DLL" << std::endl;
            FreeLibrary(hDll);
            hDll = nullptr;
            return false;
        }

        return true;
    }

    std::string extractFile(const std::string& filePath, bool enableOcr = false) {
        if (!TextMiner_ExtractFile) {
            return R"({"status":"failed","error_message":"TextMiner_ExtractFile function not loaded"})";
        }

        char* jsonResult = TextMiner_ExtractFile(filePath.c_str(), enableOcr ? 1 : 0);
        std::string result = jsonResult ? jsonResult : "";
        
        if (jsonResult) {
            free(jsonResult);
        }

        return result;
    }

    ~TextMinerWrapper() {
        if (hDll) {
            FreeLibrary(hDll);
        }
    }
};

void saveToFile(const std::string& fileName, const std::string& content) {
    std::string outputFileName = fileName + ".txt";
    std::ofstream outFile(outputFileName);
    if (outFile.is_open()) {
        outFile << content;
        outFile.close();
        std::cout << "Content saved to: " << outputFileName << std::endl;
    } else {
        std::cerr << "Failed to save file: " << outputFileName << std::endl;
    }
}

std::string extractField(const std::string& jsonStr, const std::string& fieldName) {
    std::string searchPattern = "\"" + fieldName + "\":\"";
    size_t pos = jsonStr.find(searchPattern);
    if (pos == std::string::npos) {
        return "";
    }
    
    size_t startPos = pos + searchPattern.length();
    size_t endPos = startPos;
    
    while (endPos < jsonStr.length() && jsonStr[endPos] != '"') {
        if (jsonStr[endPos] == '\\' && endPos + 1 < jsonStr.length()) {
            endPos += 2;
        } else {
            endPos++;
        }
    }
    
    return jsonStr.substr(startPos, endPos - startPos);
}

std::string extractContent(const std::string& jsonStr) {
    std::string searchPattern = "\"content\":\"";
    size_t pos = jsonStr.find(searchPattern);
    if (pos == std::string::npos) {
        return "";
    }
    
    size_t startPos = pos + searchPattern.length();
    size_t endPos = jsonStr.length() - 1;
    
    size_t bracePos = jsonStr.rfind("\"}", endPos);
    if (bracePos != std::string::npos) {
        endPos = bracePos;
    }
    
    return jsonStr.substr(startPos, endPos - startPos);
}

int main() {
    std::cout << "TextMiner DLL C++ Example" << std::endl;
    std::cout << "==================" << std::endl << std::endl;

    TextMinerWrapper textMiner;

    std::string dllPath = "TextMiner.dll";
    std::cout << "Loading DLL: " << dllPath << std::endl;
    if (!textMiner.load(dllPath)) {
        std::cerr << "Failed to initialize TextMiner wrapper" << std::endl;
        return 1;
    }
    std::cout << "DLL loaded successfully" << std::endl << std::endl;

    std::string testFile = "test.docx";
    std::cout << "Enter file path to extract (default: " << testFile << "): ";
    std::string filePath;
    std::getline(std::cin, filePath);
    if (filePath.empty()) {
        filePath = testFile;
    }

    bool enableOcr = false;
    std::cout << "Enable OCR? (y/n, default: n): ";
    std::string ocrInput;
    std::getline(std::cin, ocrInput);
    enableOcr = (ocrInput == "y" || ocrInput == "Y");

    std::cout << std::endl;
    std::cout << "Extracting file: " << filePath << std::endl;
    std::cout << "OCR enabled: " << (enableOcr ? "Yes" : "No") << std::endl << std::endl;

    std::string jsonResult = textMiner.extractFile(filePath, enableOcr);

    std::cout << "JSON Result:" << std::endl;
    std::cout << "============" << std::endl;
    std::cout << jsonResult.substr(0, 1000) << std::endl;
    if (jsonResult.length() > 1000) {
        std::cout << "..." << std::endl;
    }
    std::cout << std::endl;

    std::string status = extractField(jsonResult, "status");
    std::string fileName = extractField(jsonResult, "file_name");
    std::string fileType = extractField(jsonResult, "file_type");
    std::string errorMessage = extractField(jsonResult, "error_message");

    std::cout << "Extraction Result:" << std::endl;
    std::cout << "=================" << std::endl;
    std::cout << "File Name: " << fileName << std::endl;
    std::cout << "File Type: " << fileType << std::endl;
    std::cout << "Status: " << status << std::endl;

    if (status == "success") {
        std::string content = extractContent(jsonResult);
        std::cout << "Content Length: " << content.length() << " characters" << std::endl;
        std::cout << std::endl;
        std::cout << "Content Preview (first 500 chars):" << std::endl;
        std::cout << "---------------------------------" << std::endl;
        std::cout << content.substr(0, 500) << std::endl;
        if (content.length() > 500) {
            std::cout << "..." << std::endl;
        }
        std::cout << std::endl;

        std::cout << "Save content to file? (y/n): ";
        std::string saveInput;
        std::getline(std::cin, saveInput);
        if (saveInput == "y" || saveInput == "Y") {
            saveToFile(fileName.empty() ? "output" : fileName, content);
        }
    } else {
        std::cout << "Error Message: " << errorMessage << std::endl;
    }

    std::cout << std::endl;
    std::cout << "Press Enter to exit...";
    std::cin.get();

    return 0;
}
