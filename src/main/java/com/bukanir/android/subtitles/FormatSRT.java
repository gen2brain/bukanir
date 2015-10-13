package com.bukanir.android.subtitles;

import java.io.BufferedReader;
import java.io.FileReader;
import java.io.IOException;


/**
 * This class represents the .SRT subtitle format
 * <br><br>
 * Copyright (c) 2012 J. David Requejo <br>
 * j[dot]david[dot]requejo[at] Gmail
 * <br><br>
 * Permission is hereby granted, free of charge, to any person obtaining a copy of this software
 * and associated documentation files (the "Software"), to deal in the Software without restriction,
 * including without limitation the rights to use, copy, modify, merge, publish, distribute,
 * sublicense, and/or sell copies of the Software, and to permit persons to whom the Software
 * is furnished to do so, subject to the following conditions:
 * <br><br>
 * The above copyright notice and this permission notice shall be included in all copies
 * or substantial portions of the Software.
 * <br><br>
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED,
 * INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR
 * PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE
 * FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR
 * OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
 * DEALINGS IN THE SOFTWARE.
 * 
 * @author J. David Requejo
 *
 */
public class FormatSRT implements TimedTextFileFormat {


	public TimedTextObject parseFile(String path) throws IOException {

		TimedTextObject tto = new TimedTextObject();
		Caption caption = new Caption();
		int captionNumber = 1;
		boolean allGood;

        StringBuilder warnings = new StringBuilder();

		//first lets load the file
		BufferedReader br = new BufferedReader(new FileReader(path));

		String line = br.readLine();
        line = line.replace("\uFEFF", ""); //remove BOM character
		int lineCounter = 0;
		try {
			while(line != null){
				line = line.trim();
				lineCounter++;
				//if its a blank line, ignore it, otherwise...
				if(!line.isEmpty()){
					allGood = false;
					//the first thing should be an increasing number
					try {
						int num = Integer.parseInt(line);
						if(num != captionNumber) {
                            throw new Exception();
                        } else {
                            allGood = true;
							captionNumber++;
						}
					} catch(Exception e) {
                        warnings.append(captionNumber + " expected at line " + lineCounter);
                        warnings.append("\n skipping to next line\n\n");
					}
					if(allGood){
						//we go to next line, here the begin and end time should be found
						try {
							lineCounter++;
							line = br.readLine().trim();
							String start = line.substring(0, 12);
							String end = line.substring(line.length()-12, line.length());
							Time time = new Time("hh:mm:ss,ms",start);
							caption.start = time;
							time = new Time("hh:mm:ss,ms",end);
							caption.end = time;
						} catch(Exception e){
                            warnings.append("incorrect time format at line "+lineCounter);
							allGood = false;
						}
					}
					if(allGood){
						//we go to next line where the caption text starts
						lineCounter++;
						line = br.readLine().trim();
                        StringBuilder text = new StringBuilder();
						while(!line.isEmpty()){
                            text.append(line+"<br />");
							line = br.readLine();
                            if(line == null) break;
                            line = line.trim();
							lineCounter++;
						}
						caption.content = text.toString();
						int key = caption.start.mseconds;
						//in case the key is already there, we increase it by a millisecond, since no duplicates are allowed
						while(tto.captions.containsKey(key)) {
                            key++;
                        }
						if(key != caption.start.mseconds) {
                            warnings.append("caption with same start time found...\n\n");
                        }
						//we add the caption.
						tto.captions.put(key, caption);
					}
					//we go to next blank
					while(line != null && !line.isEmpty()) {
						line = br.readLine();
                        if(line == null) break;
                        line = line.trim();
						lineCounter++;
					}
					caption = new Caption();
				}
				line = br.readLine();
			}

		} catch(NullPointerException e){
            warnings.append("unexpected end of file, maybe last caption is not complete.\n\n");
		} finally {
	       br.close();
        }

        tto.warnings = warnings.toString();
		
		return tto;
	}

}
