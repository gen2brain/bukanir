package com.bukanir.android.providers;

import android.app.SearchManager;
import android.content.ContentProvider;
import android.content.ContentValues;
import android.content.UriMatcher;
import android.database.Cursor;
import android.database.MatrixCursor;
import android.net.Uri;
import android.provider.BaseColumns;
import android.support.annotation.NonNull;

import com.bukanir.android.clients.BukanirClient;
import com.bukanir.android.entities.AutoComplete;

import java.util.ArrayList;

public class SuggestionProvider extends ContentProvider {

    private static final int SEARCH_SUGGESTIONS = 1;

    private static final UriMatcher uriMatcher;

    static {
        uriMatcher = new UriMatcher(UriMatcher.NO_MATCH);
        uriMatcher.addURI("*", SearchManager.SUGGEST_URI_PATH_QUERY, SEARCH_SUGGESTIONS);
        uriMatcher.addURI("*", SearchManager.SUGGEST_URI_PATH_QUERY + "/*", SEARCH_SUGGESTIONS);
    }

    private static final String[] COLUMNS = new String[] {
            BaseColumns._ID,
            SearchManager.SUGGEST_COLUMN_TEXT_1,
            SearchManager.SUGGEST_COLUMN_TEXT_2,
            SearchManager.SUGGEST_COLUMN_INTENT_DATA,
            SearchManager.SUGGEST_COLUMN_INTENT_EXTRA_DATA,
    };

    public MatrixCursor cursor = new MatrixCursor(COLUMNS);

    @Override
    public boolean onCreate() {
        return true;
    }

    @Override
    public Cursor query(@NonNull Uri uri, String[] projectionIn, String selection, String[] selectionArgs, String sort) {
        int match = uriMatcher.match(uri);
        switch(match) {
            case SEARCH_SUGGESTIONS:
                String query = uri.getLastPathSegment().toLowerCase();

                ArrayList<AutoComplete> list = BukanirClient.getAutoComplete(query, 13);
                if(list != null) {
                    for(int i=0; i<list.size(); i++) {
                        AutoComplete item = list.get(i);
                        cursor.addRow(new String[]{String.valueOf(i), item.title, item.year, uri.toString(), item.title});
                    }
                }

                MatrixCursor returnMatrix = cursor;
                cursor = new MatrixCursor(COLUMNS);

                return returnMatrix;
            default:
                throw new IllegalArgumentException("Unknown URL: " + uri);
        }
    }

    @Override
    public String getType(@NonNull Uri uri) {
        return null;
    }

    @Override
    public void shutdown() {
        super.shutdown();
        cursor.close();
    }

    @Override
    public int update(@NonNull Uri uri, ContentValues values, String where, String[] whereArgs) {
        throw new UnsupportedOperationException("update not supported");
    }

    @Override
    public Uri insert(@NonNull Uri uri, ContentValues initialValues) {
        throw new UnsupportedOperationException("insert not supported");
    }

    @Override
    public int delete(@NonNull Uri uri, String where, String[] whereArgs) {
        throw new UnsupportedOperationException("delete not supported");
    }
}
