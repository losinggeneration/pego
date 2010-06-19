(function() {
	// Mapping of language -> class name -> list of string/regex
	var languages = {
		"c": {
			"keyword": [
				"auto", "break", "case", "char", "const", "continue", "default", "do",
				"double", "else", "enum", "extern", "float", "for", "goto", "if", "int",
				"long", "register", "return", "short", "signed", "sizeof", "static",
				"struct", "switch", "typedef", "union", "unsigned", "void", "volatile",
				"while"
			],
			"separator": [
				"~", "}", "||", "|=", "|", "{", "^=", "^", "]", "[", "?", ">>=", ">>", ">=",
				">", "==", "=", "<=", "<<=", "<<", "<", ";", ":", "/=", "/", "...", ".",
				"->", "-=", "--", "-", ",", "+=", "++", "+", "*=", "*", ")", "(", "&=",
				"&&", "&", "%=", "%", "##", "#", "!=", "!"
			],
			"type": [
				"auto", "char", "const", "double", "extern", "float", "int", "int16_t",
				"int32_t", "int64_t", "int8_t", "long", "register", "short", "signed",
				"uint16_t", "uint32_t", "uint64_t", "uint8_t", "unsigned", "volatile"
			],
			"value": [
				/"(?:\\.|[^"\\])*"/,                     // string literal
				/'(?:\\.|[^"\\])*'/,                     // char literal
				/(?:\b\d+\.\d+|\.\d+)(?:E[+-]?\d+)?\b/i, // floating point literal (1)
				/\b\d+(?:\.?E[+-]?\d+\b|\.)/i,           // floating point literal (2)
				/\b[1-9]\d+\b/,                          // decimal integer literal
				/\b0x[0-9a-f]+\b/i,                      // hexadecimal integer literal
				/\b0[0-7]*\b/                            // octal integer literal
			],
			"comment": [
				/\/\*[\s\S]*?\*\//, // Multi-line comment
				/\/\/.*/,           // Single-line comment
				/#.*(?:\\\n.*)/     // Pre-processor instruction
			]
		},
		"lua": {
			"keyword": [
				"and", "break", "do", "else", "elseif", "end", "for", "function", "if",
				"in", "local", "not", "or", "repeat", "return", "then", "until", "while"
			],
			"separator": [
				"~=", "}", "{", "^", "]", "[", ">=", ">", "==", "=", "<=", "<", ";", ":",
				"/", "...", "..", ".", "-", ",", "+", "*", ")", "(", "%", "#"
			],
			"value": [
				/"(?:\\.|\\\n|[^\\"])*"/,
				/'(?:\\.|\\\n|[^\\"])*'/,
				/\[(=*)\[(?:.|\n)*?]\1]/,
				/\b\d\+(?:\.\d+)?(?:E[+-]?\d+)?\b/i,
				/\b0x[0-9a-f]+\b/i
			],
			"comment": [
				/--(?!\[=*\[).*/,
				/--\[(=*)\[(?:.|\n)*?]\1]/
			]
		},
		"go": {
			"keyword": [
				"break", "default", "func", "interface", "select", "case", "defer", "go",
				"map", "struct", "chan", "else", "goto", "package", "switch", "const",
				"fallthrough", "if", "range", "type", "continue", "for", "import", "return",
				"var"
			],
			"type": [
				"uintptr", "uint8", "uint64", "uint32", "uint16", "uint", "string", "int8",
				"int64", "int32", "int16", "int", "float64", "float32", "float",
				"complex64", "complex128", "complex", "byte"
			],
			"separator": [
				"}", "||", "|=", "|", "{", "^=", "^", "]", "[", ">>=", ">>", ">=", ">",
				"==", "=", "<=", "<<=", "<<", "<-", "<", ";", ":=", ":", "/=", "/", "...",
				".", "-=", "--", "-", ",", "+=", "++", "+", "*=", "*", ")", "(", "&^=", "&^",
				"&=", "&&", "&", "%=", "%", "!=", "!",
			],
			"value": [
				/"(?:\\.|[^\\"])*"/,                            // string literal
				/'(?:\\.|[^\\'])+'/,                            // character literal
				/`[^`]*`/,                                      // raw string literal
				/\b(?:[1-9]\d*i?|0i?|0x[0-9a-f]+|0[0-7]+)\b/i,  // integer literal
				/(?:\b\d+\.\d+|\.\d+)(?:E[+-]?\d+)?i?\b/i,      // floating point literal (1)
				/\b\d+(?:\.?E[+-]?\d+i?\b|\.i\b|\.)/i,          // floating point literal (2)
			],
			"comment": [
				/\/\*[\s\S]*?\*\//, // Multi-line comment
				/\/\/.*/            // Single-line comment
			]
		}
	};

	var escapeRegexp = function(s) {
		var meta = /[\\^$*+?.()|{[]/g;
		var escaper = function(match) {
			return "\\" + match;
		};
		return s.replace(meta, escaper);
	}

	var compile = function(strings) {
		var regex = [];
		strings.sort(function(a,b) {
			// longest first
			if (a.length < b.length) {
				return 1;
			}
			else if (a.length > b.length) {
				return -1;
			}
			else {
				return 0;
			}
		});
		for (var i = 0; i < strings.length; i++) {
			var p = escapeRegexp(strings[i]);
			if (/^\w+$/.test(strings[i])) {
				p = "\\b" + p + "\\b";
			}
			regex.push(p);
		}
		return new RegExp(regex.join("|"), "g");
	}

	var patcmp = function(a, b) {
		// first then longest
		if (a.index < b.index) {
			return -1;
		}
		else if (a.index > b.index) {
			return 1;
		}
		else if (a.size > b.size) {
			return -1;
		}
		else if (a.size < b.size) {
			return 1;
		}
		else {
			return 0;
		}
	}

	var rebuildRegExp = function(source, ignoreCase, multiLine) {
		var flags = "g";
		if (ignoreCase) { flags += "i"; }
		if (multiLine) { flags += "m"; }
		return new RegExp(source, flags);
	}

	var highlight = function(name, node, patterns) {
		if (node.nodeType == 3) {
			var text = node.data;
			//~console.log("Highlighting:", text);
			var positions = [];
			for (var i = 0; i < patterns.length; i++) {
				var e = patterns[i];
				e.re.lastIndex = 0;
				var match = e.re.exec(text);
				if (match == null) {
					//~console.log("Could not find any", name, e.name, e.count, ":", e.re);
					continue;
				}
				positions.push({
					"re":    e.re,
					"name":  e.name,
					"count": e.count,
					"index": match.index,
					"size":  match[0].length
				});
			}
			var offset = 0; // position of last split
			var pos = 0; // start of next search
			var count = 0;
			while (positions.length > 0) {
				positions.sort(patcmp);
				var e = positions[0];
				if (count > 10) {
					//~console.log("aborting...");
					//~console.log("regex:", e.re);
					break;
				}
				count++;
				//~console.log("Highlighting", name, e.name, e.count, ":", text.substr(e.index, e.size), "( i:", e.index, "s:", e.size, ")");
				if (e.size > 0) {
					var middleBit;
					if (e.index > offset) {
						middleBit = node.splitText(e.index-offset);
					}
					else {
						middleBit = node;
					}
					var endBit = middleBit.splitText(e.size);
					var span = document.createElement("span");
					span.className = e.name;
					span.appendChild(middleBit.cloneNode(true));
					middleBit.parentNode.replaceChild(span, middleBit);
					node = endBit;
					offset = e.index + e.size;
				}
				pos = e.index + e.size;
				if (e.size == 0) { pos++; }
				for (var i = 0; i < positions.length; i++) {
					var e = positions[i];
					if (e.index >= offset) { break }
					e.re.global = true;
					e.re.lastIndex = pos;
					var match = e.re.exec(text);
					if (match == null) {
						positions.splice(i,1);
						i--;
					}
					else {
						//~console.log("from", e.index, pos, "next", e.name, e.count, "at", match.index, ":", match[0]);
						positions[i].index = match.index;
						positions[i].size = match[0].length;
					}
				}
			}
		}
		else if (node.nodeType == 1 && node.childNodes &&
				!/^(?:script|style)$/i.test(node.tagName)) {
			var children = node.childNodes;
			for (var i = 0; i < children.length; i++) {
				highlight(name, children[i], patterns);
			}
		}
	}

	var prepareCache = {}
	var prepare = function(name, rules) {
		var patterns = prepareCache[name];
		if (patterns !== undefined) {
			return patterns;
		}
		patterns = [];
		for (var key in rules) {
			if (!rules.hasOwnProperty(key) || rules[key].length == 0) {
				continue;
			}
			var values = rules[key];
			var strings = [];
			var regexes = [];
			for (var j = 0; j < values.length; j++) {
				var v = values[j];
				if (typeof v == "string" || v instanceof String) {
					strings.push(v);
				}
				else if (v instanceof RegExp) {
					regexes.push(v);
				}
			}
			count = 1;
			if (strings.length > 0) {
				patterns.push({
					"re": compile(strings),
					"name": key,
					"count": count,
					"index": -1,
					"size": -1
				});
				count++;
			}
			for (var j = 0; j < regexes.length; j++) {
				var re = regexes[j];
				if (!re.global) {
					re.global = true;
					if (!re.global) {
						// Unable to modify flags (Google Chrome)
						re = rebuildRegExp(re.source,re.ignoreCase,re.multiline);
					}
				}
				patterns.push({
					"re": re,
					"name": key,
					"count": count,
					"index": -1,
					"size": -1
				});
				count++;
			}
		}
		prepareCache[name] = patterns;
		return patterns;
	}

	var HighlightNode = function(node, lang) {
		if (languages[lang] === undefined) { return; }
		var patterns = prepare(lang,languages[lang]);
		highlight(lang, node, patterns);
	}

	window.HighlightNode = HighlightNode;
})();
