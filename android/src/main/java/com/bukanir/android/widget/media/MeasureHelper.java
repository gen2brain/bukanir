/*
 * Copyright (C) 2015 Zhang Rui <bbcallen@gmail.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package com.bukanir.android.widget.media;

import android.content.Context;
import android.support.annotation.NonNull;
import android.view.View;

import com.bukanir.android.R;

import java.lang.ref.WeakReference;

public final class MeasureHelper {
    private WeakReference<View> weakView;

    private int videoWidth;
    private int videoHeight;
    private int videoSarNum;
    private int videoSarDen;

    private int videoRotationDegree;

    private int measuredWidth;
    private int measuredHeight;

    private int currentAspectRatio = IRenderView.AR_ASPECT_FIT_PARENT;

    public MeasureHelper(View view) {
        weakView = new WeakReference<>(view);
    }

    public View getView() {
        if(weakView == null)
            return null;
        return weakView.get();
    }

    public void setVideoSize(int videoWidth, int videoHeight) {
        this.videoWidth = videoWidth;
        this.videoHeight = videoHeight;
    }

    public void setVideoSampleAspectRatio(int videoSarNum, int videoSarDen) {
        this.videoSarNum = videoSarNum;
        this.videoSarDen = videoSarDen;
    }

    public void setVideoRotation(int videoRotationDegree) {
        this.videoRotationDegree = videoRotationDegree;
    }

    /**
     * Must be called by View.onMeasure(int, int)
     *
     * @param widthMeasureSpec
     * @param heightMeasureSpec
     */
    public void doMeasure(int widthMeasureSpec, int heightMeasureSpec) {
        if(videoRotationDegree == 90 || videoRotationDegree == 270) {
            int tempSpec = widthMeasureSpec;
            widthMeasureSpec  = heightMeasureSpec;
            heightMeasureSpec = tempSpec;
        }

        int width = View.getDefaultSize(videoWidth, widthMeasureSpec);
        int height = View.getDefaultSize(videoHeight, heightMeasureSpec);
        if(currentAspectRatio == IRenderView.AR_MATCH_PARENT) {
            width = widthMeasureSpec;
            height = heightMeasureSpec;
        } else if (videoWidth > 0 && videoHeight > 0) {
            int widthSpecMode = View.MeasureSpec.getMode(widthMeasureSpec);
            int widthSpecSize = View.MeasureSpec.getSize(widthMeasureSpec);
            int heightSpecMode = View.MeasureSpec.getMode(heightMeasureSpec);
            int heightSpecSize = View.MeasureSpec.getSize(heightMeasureSpec);

            if(widthSpecMode == View.MeasureSpec.AT_MOST && heightSpecMode == View.MeasureSpec.AT_MOST) {
                float specAspectRatio = (float) widthSpecSize / (float) heightSpecSize;
                float displayAspectRatio;
                switch(currentAspectRatio) {
                    case IRenderView.AR_16_9_FIT_PARENT:
                        displayAspectRatio = 16.0f / 9.0f;
                        if(videoRotationDegree == 90 || videoRotationDegree == 270)
                            displayAspectRatio = 1.0f / displayAspectRatio;
                        break;
                    case IRenderView.AR_4_3_FIT_PARENT:
                        displayAspectRatio = 4.0f / 3.0f;
                        if(videoRotationDegree == 90 || videoRotationDegree == 270)
                            displayAspectRatio = 1.0f / displayAspectRatio;
                        break;
                    case IRenderView.AR_ASPECT_FIT_PARENT:
                    case IRenderView.AR_ASPECT_FILL_PARENT:
                    case IRenderView.AR_ASPECT_WRAP_CONTENT:
                    default:
                        displayAspectRatio = (float) videoWidth / (float) videoHeight;
                        if(videoSarNum > 0 && videoSarDen > 0)
                            displayAspectRatio = displayAspectRatio * videoSarNum / videoSarDen;
                        break;
                }
                boolean shouldBeWider = displayAspectRatio > specAspectRatio;

                switch(currentAspectRatio) {
                    case IRenderView.AR_ASPECT_FIT_PARENT:
                    case IRenderView.AR_16_9_FIT_PARENT:
                    case IRenderView.AR_4_3_FIT_PARENT:
                        if(shouldBeWider) {
                            // too wide, fix width
                            width = widthSpecSize;
                            height = (int) (width / displayAspectRatio);
                        } else {
                            // too high, fix height
                            height = heightSpecSize;
                            width = (int) (height * displayAspectRatio);
                        }
                        break;
                    case IRenderView.AR_ASPECT_FILL_PARENT:
                        if(shouldBeWider) {
                            // not high enough, fix height
                            height = heightSpecSize;
                            width = (int) (height * displayAspectRatio);
                        } else {
                            // not wide enough, fix width
                            width = widthSpecSize;
                            height = (int) (width / displayAspectRatio);
                        }
                        break;
                    case IRenderView.AR_ASPECT_WRAP_CONTENT:
                    default:
                        if(shouldBeWider) {
                            // too wide, fix width
                            width = Math.min(videoWidth, widthSpecSize);
                            height = (int) (width / displayAspectRatio);
                        } else {
                            // too high, fix height
                            height = Math.min(videoHeight, heightSpecSize);
                            width = (int) (height * displayAspectRatio);
                        }
                        break;
                }
            } else if(widthSpecMode == View.MeasureSpec.EXACTLY && heightSpecMode == View.MeasureSpec.EXACTLY) {
                // the size is fixed
                width = widthSpecSize;
                height = heightSpecSize;

                // for compatibility, we adjust size based on aspect ratio
                if(videoWidth * height < width * videoHeight) {
                    width = height * videoWidth / videoHeight;
                } else if(videoWidth * height > width * videoHeight) {
                    height = width * videoHeight / videoWidth;
                }
            } else if(widthSpecMode == View.MeasureSpec.EXACTLY) {
                // only the width is fixed, adjust the height to match aspect ratio if possible
                width = widthSpecSize;
                height = width * videoHeight / videoWidth;
                if (heightSpecMode == View.MeasureSpec.AT_MOST && height > heightSpecSize) {
                    // couldn't match aspect ratio within the constraints
                    height = heightSpecSize;
                }
            } else if(heightSpecMode == View.MeasureSpec.EXACTLY) {
                // only the height is fixed, adjust the width to match aspect ratio if possible
                height = heightSpecSize;
                width = height * videoWidth / videoHeight;
                if(widthSpecMode == View.MeasureSpec.AT_MOST && width > widthSpecSize) {
                    // couldn't match aspect ratio within the constraints
                    width = widthSpecSize;
                }
            } else {
                // neither the width nor the height are fixed, try to use actual video size
                width = videoWidth;
                height = videoHeight;
                if(heightSpecMode == View.MeasureSpec.AT_MOST && height > heightSpecSize) {
                    // too tall, decrease both width and height
                    height = heightSpecSize;
                    width = height * videoWidth / videoHeight;
                }
                if(widthSpecMode == View.MeasureSpec.AT_MOST && width > widthSpecSize) {
                    // too wide, decrease both width and height
                    width = widthSpecSize;
                    height = width * videoHeight / videoWidth;
                }
            }
        } else {
            // no size yet, just adopt the given spec sizes
        }

        measuredWidth = width;
        measuredHeight = height;
    }

    public int getMeasuredWidth() {
        return measuredWidth;
    }

    public int getMeasuredHeight() {
        return measuredHeight;
    }

    public void setAspectRatio(int aspectRatio) {
        currentAspectRatio = aspectRatio;
    }

    @NonNull
    public static String getAspectRatioText(Context context, int aspectRatio) {
        String text;
        switch(aspectRatio) {
            case IRenderView.AR_ASPECT_FIT_PARENT:
                text = context.getString(R.string.VideoView_ar_aspect_fit_parent);
                break;
            case IRenderView.AR_ASPECT_FILL_PARENT:
                text = context.getString(R.string.VideoView_ar_aspect_fill_parent);
                break;
            case IRenderView.AR_ASPECT_WRAP_CONTENT:
                text = context.getString(R.string.VideoView_ar_aspect_wrap_content);
                break;
            case IRenderView.AR_MATCH_PARENT:
                text = context.getString(R.string.VideoView_ar_match_parent);
                break;
            case IRenderView.AR_16_9_FIT_PARENT:
                text = context.getString(R.string.VideoView_ar_16_9_fit_parent);
                break;
            case IRenderView.AR_4_3_FIT_PARENT:
                text = context.getString(R.string.VideoView_ar_4_3_fit_parent);
                break;
            default:
                text = context.getString(R.string.N_A);
                break;
        }
        return text;
    }
}
