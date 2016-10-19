install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libswresample.2.dylib @executable_path/../Frameworks/libswresample.2.dylib libavcodec.57.dylib
install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libavutil.55.dylib @executable_path/../Frameworks/libavutil.55.dylib libavcodec.57.dylib

install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libavfilter.6.dylib @executable_path/../Frameworks/libavfilter.6.dylib libavdevice.57.dylib
install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libswscale.4.dylib @executable_path/../Frameworks/libswscale.4.dylib libavdevice.57.dylib
install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libpostproc.54.dylib @executable_path/../Frameworks/libpostproc.54.dylib libavdevice.57.dylib
install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libavresample.3.dylib @executable_path/../Frameworks/libavresample.3.dylib libavdevice.57.dylib
install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libavformat.57.dylib @executable_path/../Frameworks/libavformat.57.dylib libavdevice.57.dylib
install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libavcodec.57.dylib @executable_path/../Frameworks/libavcodec.57.dylib libavdevice.57.dylib
install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libswresample.2.dylib @executable_path/../Frameworks/libswresample.2.dylib libavdevice.57.dylib
install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libavutil.55.dylib @executable_path/../Frameworks/libavutil.55.dylib libavdevice.57.dylib

install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libavfilter.6.dylib @executable_path/../Frameworks/libavfilter.6.dylib libavfilter.6.dylib
install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libswscale.4.dylib @executable_path/../Frameworks/libswscale.4.dylib libavfilter.6.dylib
install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libpostproc.54.dylib @executable_path/../Frameworks/libpostproc.54.dylib libavfilter.6.dylib
install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libavresample.3.dylib @executable_path/../Frameworks/libavresample.3.dylib libavfilter.6.dylib
install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libavformat.57.dylib @executable_path/../Frameworks/libavformat.57.dylib libavfilter.6.dylib
install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libavcodec.57.dylib @executable_path/../Frameworks/libavcodec.57.dylib libavfilter.6.dylib
install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libswresample.2.dylib @executable_path/../Frameworks/libswresample.2.dylib libavfilter.6.dylib
install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libavutil.55.dylib @executable_path/../Frameworks/libavutil.55.dylib libavfilter.6.dylib

install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libavcodec.57.dylib @executable_path/../Frameworks/libavcodec.57.dylib libavformat.57.dylib
install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libswresample.2.dylib @executable_path/../Frameworks/libswresample.2.dylib libavformat.57.dylib
install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libavutil.55.dylib @executable_path/../Frameworks/libavutil.55.dylib libavformat.57.dylib

install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libavutil.55.dylib @executable_path/../Frameworks/libavutil.55.dylib libavresample.3.dylib

install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libavutil.55.dylib @executable_path/../Frameworks/libavutil.55.dylib libpostproc.54.dylib

install_name_tool -change /usr/local/Cellar/openssl/1.0.2j/lib/libcrypto.1.0.0.dylib @executable_path/../Frameworks/libcrypto.1.0.0.dylib libssl.1.0.0.dylib

install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libavutil.55.dylib @executable_path/../Frameworks/libavutil.55.dylib libswresample.2.dylib

install_name_tool -change /usr/local/Cellar/ffmpeg/3.1.4/lib/libavutil.55.dylib @executable_path/../Frameworks/libavutil.55.dylib libswscale.4.dylib

cp -r /System/Library/Frameworks/VideoToolbox.framework ./
cp -r /System/Library/Frameworks/AudioToolbox.framework ./
install_name_tool -change /System/Library/Frameworks/VideoToolbox.framework/Versions/A/VideoToolbox @executable_path/../Frameworks/VideoToolbox.framework/Versions/A/VideoToolbox libavcodec.57.dylib
install_name_tool -change /System/Library/Frameworks/AudioToolbox.framework/Versions/A/AudioToolbox @executable_path/../Frameworks/AudioToolbox.framework/Versions/A/AudioToolbox libavcodec.57.dylib
